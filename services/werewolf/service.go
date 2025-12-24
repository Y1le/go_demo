package werewolf

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	pb "liam/pkg/werewolf"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type WerewolfServer struct {
	pb.UnimplementedWerewolfServiceServer
	rooms map[string]*GameRoom
	mu    sync.RWMutex
}

type GameRoom struct {
	ID           string
	Name         string
	MaxPlayers   int
	Players      map[string]*pb.Player
	State        pb.GameState
	CurrentPhase pb.Phase
	DayCount     int
	RoleConfig   map[string]int32

	// 夜晚行动记录
	NightActions      map[string]*pb.NightAction
	GuardTarget       string // 守卫保护的目标
	WerewolfTarget    string // 狼人击杀的目标
	WitchSaveUsed     bool   // 女巫是否用过解药
	WitchPoisonUsed   bool   // 女巫是否用过毒药
	WitchSaveTarget   string // 女巫救人目标
	WitchPoisonTarget string // 女巫毒人目标

	// 投票记录
	Votes       map[string]string // voter_id -> target_id
	DeadPlayers map[string]bool

	// 事件订阅
	Subscribers map[string]chan *pb.GameEvent

	// 阶段控制
	PhaseTimer *time.Timer
	PhaseDone  chan bool

	mu sync.RWMutex
}

func NewWerewolfServer() *WerewolfServer {
	return &WerewolfServer{
		rooms: make(map[string]*GameRoom),
	}
}

// CreateRoom 创建游戏房间
func (s *WerewolfServer) CreateRoom(ctx context.Context, req *pb.CreateRoomRequest) (*pb.CreateRoomResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	roomID := generateRoomID()
	room := &GameRoom{
		ID:           roomID,
		Name:         req.RoomName,
		MaxPlayers:   int(req.MaxPlayers),
		Players:      make(map[string]*pb.Player),
		State:        pb.GameState_WAITING,
		CurrentPhase: pb.Phase_PHASE_WAITING,
		RoleConfig:   req.RoleConfig,
		DeadPlayers:  make(map[string]bool),
		Votes:        make(map[string]string),
		NightActions: make(map[string]*pb.NightAction),
		Subscribers:  make(map[string]chan *pb.GameEvent),
		PhaseDone:    make(chan bool, 1),
	}

	s.rooms[roomID] = room

	return &pb.CreateRoomResponse{
		RoomId:  roomID,
		Message: fmt.Sprintf("房间 %s 创建成功", req.RoomName),
	}, nil
}

// JoinRoom 加入房间
func (s *WerewolfServer) JoinRoom(ctx context.Context, req *pb.JoinRoomRequest) (*pb.JoinRoomResponse, error) {
	s.mu.RLock()
	room, exists := s.rooms[req.RoomId]
	s.mu.RUnlock()

	if !exists {
		return nil, status.Error(codes.NotFound, "房间不存在")
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if len(room.Players) >= room.MaxPlayers {
		return &pb.JoinRoomResponse{
			Success: false,
			Message: "房间已满",
		}, nil
	}

	if room.State != pb.GameState_WAITING {
		return &pb.JoinRoomResponse{
			Success: false,
			Message: "游戏已开始，无法加入",
		}, nil
	}

	player := &pb.Player{
		PlayerId: req.PlayerId,
		Name:     req.PlayerName,
		Role:     pb.Role_UNKNOWN,
		Camp:     pb.Camp_CAMP_UNKNOWN,
		IsAlive:  true,
		Position: int32(len(room.Players) + 1),
		CanAct:   false,
	}

	room.Players[req.PlayerId] = player

	// 广播玩家加入事件
	room.broadcastEvent(&pb.GameEvent{
		EventType: pb.GameEvent_EVENT_PLAYER_JOINED,
		Message:   fmt.Sprintf("%s 加入了房间", req.PlayerName),
		Timestamp: time.Now().Unix(),
	})

	return &pb.JoinRoomResponse{
		Success: true,
		Message: "加入房间成功",
		Player:  player,
	}, nil
}

// StartGame 开始游戏
func (s *WerewolfServer) StartGame(ctx context.Context, req *pb.StartGameRequest) (*pb.StartGameResponse, error) {
	s.mu.RLock()
	room, exists := s.rooms[req.RoomId]
	s.mu.RUnlock()

	if !exists {
		return nil, status.Error(codes.NotFound, "房间不存在")
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if room.State != pb.GameState_WAITING {
		return &pb.StartGameResponse{
			Success: false,
			Message: "游戏已经开始",
		}, nil
	}

	// 分配角色
	if err := assignRoles(room); err != nil {
		return &pb.StartGameResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	room.State = pb.GameState_NIGHT
	room.DayCount = 1
	room.CurrentPhase = pb.Phase_PHASE_NIGHT_GUARD

	// 广播游戏开始事件
	room.broadcastEvent(&pb.GameEvent{
		EventType: pb.GameEvent_EVENT_GAME_STARTED,
		Message:   "游戏开始！天黑请闭眼...",
		PhaseInfo: room.getCurrentPhaseInfo(),
		Timestamp: time.Now().Unix(),
	})

	// 启动游戏流程控制器
	go room.runGameLoop()

	return &pb.StartGameResponse{
		Success:   true,
		Message:   "游戏开始，第一个黑夜",
		PhaseInfo: room.getCurrentPhaseInfo(),
	}, nil
}

// runGameLoop 游戏主循环 - 上帝视角控制
func (room *GameRoom) runGameLoop() {
	for {
		room.mu.Lock()

		// 检查游戏是否结束
		if winner := room.checkGameOver(); winner != pb.Camp_CAMP_UNKNOWN {
			room.State = pb.GameState_FINISHED
			room.CurrentPhase = pb.Phase_PHASE_GAME_OVER

			room.broadcastEvent(&pb.GameEvent{
				EventType: pb.GameEvent_EVENT_GAME_OVER,
				Message:   fmt.Sprintf("游戏结束！%s 阵营获胜", getCampName(winner)),
				Timestamp: time.Now().Unix(),
			})

			room.mu.Unlock()
			return
		}

		currentPhase := room.CurrentPhase
		room.mu.Unlock()

		// 执行当前阶段
		switch currentPhase {
		case pb.Phase_PHASE_NIGHT_GUARD:
			room.executeGuardPhase()
		case pb.Phase_PHASE_NIGHT_WEREWOLF:
			room.executeWerewolfPhase()
		case pb.Phase_PHASE_NIGHT_WITCH:
			room.executeWitchPhase()
		case pb.Phase_PHASE_NIGHT_SEER:
			room.executeSeerPhase()
		case pb.Phase_PHASE_DAY_DISCUSSION:
			room.executeDayDiscussion()
		case pb.Phase_PHASE_DAY_VOTING:
			room.executeVotingPhase()
		case pb.Phase_PHASE_DAY_LAST_WORDS:
			room.executeLastWords()
		}

		// 等待阶段完成或超时
		select {
		case <-room.PhaseDone:
			// 阶段完成，继续下一阶段
		case <-time.After(60 * time.Second):
			// 超时，强制进入下一阶段
			log.Printf("阶段 %v 超时", currentPhase)
		}

		// 进入下一阶段
		room.mu.Lock()
		room.nextPhase()
		room.mu.Unlock()
	}
}

// executeGuardPhase 守卫阶段
func (room *GameRoom) executeGuardPhase() {
	room.mu.Lock()
	defer room.mu.Unlock()

	log.Printf("房间 %s: 进入守卫阶段", room.ID)

	// 找出守卫
	var guard *pb.Player
	for _, player := range room.Players {
		if player.Role == pb.Role_GUARD && player.IsAlive {
			guard = player
			break
		}
	}

	if guard == nil {
		// 没有守卫或守卫已死，跳过
		room.PhaseDone <- true
		return
	}

	// 通知守卫行动
	guard.CanAct = true
	room.broadcastEvent(&pb.GameEvent{
		EventType: pb.GameEvent_EVENT_YOUR_TURN,
		Message:   "守卫请睁眼，选择你要保护的人",
		PhaseInfo: room.getCurrentPhaseInfo(),
		Timestamp: time.Now().Unix(),
		ExtraData: map[string]string{
			"target_player_id": guard.PlayerId,
		},
	})

	// 重置守卫目标
	room.GuardTarget = ""
}

// executeWerewolfPhase 狼人阶段
func (room *GameRoom) executeWerewolfPhase() {
	room.mu.Lock()
	defer room.mu.Unlock()

	log.Printf("房间 %s: 进入狼人阶段", room.ID)

	// 找出所有存活的狼人
	werewolves := make([]*pb.Player, 0)
	for _, player := range room.Players {
		if player.Role == pb.Role_WEREWOLF && player.IsAlive {
			werewolves = append(werewolves, player)
			player.CanAct = true
		}
	}

	if len(werewolves) == 0 {
		room.PhaseDone <- true
		return
	}

	// 通知狼人行动
	room.broadcastEvent(&pb.GameEvent{
		EventType:       pb.GameEvent_EVENT_YOUR_TURN,
		Message:         "狼人请睁眼，选择你要击杀的对象",
		PhaseInfo:       room.getCurrentPhaseInfo(),
		AffectedPlayers: werewolves,
		Timestamp:       time.Now().Unix(),
	})

	// 重置狼人目标
	room.WerewolfTarget = ""
}

// executeWitchPhase 女巫阶段
func (room *GameRoom) executeWitchPhase() {
	room.mu.Lock()
	defer room.mu.Unlock()

	log.Printf("房间 %s: 进入女巫阶段", room.ID)

	// 找出女巫
	var witch *pb.Player
	for _, player := range room.Players {
		if player.Role == pb.Role_WITCH && player.IsAlive {
			witch = player
			break
		}
	}

	if witch == nil {
		room.PhaseDone <- true
		return
	}

	witch.CanAct = true

	// 判断今晚是否有人被杀且未被守卫保护
	victimID := ""
	if room.WerewolfTarget != "" && room.WerewolfTarget != room.GuardTarget {
		victimID = room.WerewolfTarget
	}

	extraData := map[string]string{
		"target_player_id": witch.PlayerId,
		"save_available":   fmt.Sprintf("%v", !room.WitchSaveUsed),
		"poison_available": fmt.Sprintf("%v", !room.WitchPoisonUsed),
	}

	message := "女巫请睁眼"
	if victimID != "" {
		extraData["victim_id"] = victimID
		message += fmt.Sprintf("，今晚 %s 号玩家死了，是否使用解药？", room.Players[victimID].Name)
	} else {
		message += "，今晚平安夜"
	}

	room.broadcastEvent(&pb.GameEvent{
		EventType: pb.GameEvent_EVENT_YOUR_TURN,
		Message:   message,
		PhaseInfo: room.getCurrentPhaseInfo(),
		Timestamp: time.Now().Unix(),
		ExtraData: extraData,
	})
}

// executeSeerPhase 预言家阶段
func (room *GameRoom) executeSeerPhase() {
	room.mu.Lock()
	defer room.mu.Unlock()

	log.Printf("房间 %s: 进入预言家阶段", room.ID)

	// 找出预言家
	var seer *pb.Player
	for _, player := range room.Players {
		if player.Role == pb.Role_SEER && player.IsAlive {
			seer = player
			break
		}
	}

	if seer == nil {
		room.PhaseDone <- true
		return
	}

	seer.CanAct = true

	room.broadcastEvent(&pb.GameEvent{
		EventType: pb.GameEvent_EVENT_YOUR_TURN,
		Message:   "预言家请睁眼，选择你要查验的人",
		PhaseInfo: room.getCurrentPhaseInfo(),
		Timestamp: time.Now().Unix(),
		ExtraData: map[string]string{
			"target_player_id": seer.PlayerId,
		},
	})
}

// executeDayDiscussion 白天讨论阶段
func (room *GameRoom) executeDayDiscussion() {
	room.mu.Lock()
	defer room.mu.Unlock()

	log.Printf("房间 %s: 进入白天讨论阶段", room.ID)

	// 结算昨晚的死亡
	deadPlayers := room.settleNightDeaths()

	message := "天亮了"
	if len(deadPlayers) > 0 {
		names := make([]string, 0)
		for _, p := range deadPlayers {
			names = append(names, fmt.Sprintf("%s(%d号)", p.Name, p.Position))
		}
		message += fmt.Sprintf("，昨晚 %s 死了", joinStrings(names, "、"))
	} else {
		message += "，昨晚是平安夜"
	}

	room.broadcastEvent(&pb.GameEvent{
		EventType:       pb.GameEvent_EVENT_PHASE_CHANGED,
		Message:         message,
		PhaseInfo:       room.getCurrentPhaseInfo(),
		AffectedPlayers: deadPlayers,
		Timestamp:       time.Now().Unix(),
	})

	// 讨论时间（可以设置为60秒）
	time.Sleep(5 * time.Second) // 简化演示，实际应该等待用户交互
	room.PhaseDone <- true
}

// executeVotingPhase 投票阶段
func (room *GameRoom) executeVotingPhase() {
	room.mu.Lock()
	defer room.mu.Unlock()

	log.Printf("房间 %s: 进入投票阶段", room.ID)

	// 所有存活玩家可以投票
	for _, player := range room.Players {
		if player.IsAlive {
			player.CanAct = true
		}
	}

	room.broadcastEvent(&pb.GameEvent{
		EventType: pb.GameEvent_EVENT_PHASE_CHANGED,
		Message:   "请所有玩家投票，选择你认为是狼人的玩家",
		PhaseInfo: room.getCurrentPhaseInfo(),
		Timestamp: time.Now().Unix(),
	})

	// 清空投票记录
	room.Votes = make(map[string]string)
}

// executeLastWords 遗言阶段
func (room *GameRoom) executeLastWords() {
	room.mu.Lock()
	defer room.mu.Unlock()

	log.Printf("房间 %s: 进入遗言阶段", room.ID)

	// 统计投票结果
	votedOut := room.countVotes()

	if votedOut != nil {
		votedOut.IsAlive = false
		room.DeadPlayers[votedOut.PlayerId] = true

		room.broadcastEvent(&pb.GameEvent{
			EventType:       pb.GameEvent_EVENT_PLAYER_DIED,
			Message:         fmt.Sprintf("%s(%d号) 被投票出局，请留遗言", votedOut.Name, votedOut.Position),
			PhaseInfo:       room.getCurrentPhaseInfo(),
			AffectedPlayers: []*pb.Player{votedOut},
			Timestamp:       time.Now().Unix(),
		})

		// 等待遗言时间
		time.Sleep(3 * time.Second)
	} else {
		room.broadcastEvent(&pb.GameEvent{
			EventType: pb.GameEvent_EVENT_PHASE_CHANGED,
			Message:   "本轮没有玩家被投票出局",
			PhaseInfo: room.getCurrentPhaseInfo(),
			Timestamp: time.Now().Unix(),
		})
	}

	room.PhaseDone <- true
}

// NightAction 夜晚行动
func (s *WerewolfServer) NightAction(ctx context.Context, req *pb.NightActionRequest) (*pb.NightActionResponse, error) {
	s.mu.RLock()
	room, exists := s.rooms[req.RoomId]
	s.mu.RUnlock()

	if !exists {
		return nil, status.Error(codes.NotFound, "房间不存在")
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if room.State != pb.GameState_NIGHT {
		return &pb.NightActionResponse{
			Success: false,
			Message: "当前不是夜晚阶段",
		}, nil
	}

	player, exists := room.Players[req.PlayerId]
	if !exists || !player.IsAlive {
		return &pb.NightActionResponse{
			Success: false,
			Message: "玩家不存在或已死亡",
		}, nil
	}

	if !player.CanAct {
		return &pb.NightActionResponse{
			Success: false,
			Message: "当前不是你的行动时间",
		}, nil
	}

	// 根据角色和阶段处理行动
	var result string
	var err error

	switch player.Role {
	case pb.Role_GUARD:
		if room.CurrentPhase == pb.Phase_PHASE_NIGHT_GUARD {
			room.GuardTarget = req.TargetPlayerId
			result = "守卫成功"
			player.CanAct = false
			room.PhaseDone <- true
		} else {
			err = errors.New("当前不是守卫阶段")
		}

	case pb.Role_WEREWOLF:
		if room.CurrentPhase == pb.Phase_PHASE_NIGHT_WEREWOLF {
			room.WerewolfTarget = req.TargetPlayerId
			result = "选择击杀目标成功"
			player.CanAct = false

			// 检查是否所有狼人都行动了
			allActed := true
			for _, p := range room.Players {
				if p.Role == pb.Role_WEREWOLF && p.IsAlive && p.CanAct {
					allActed = false
					break
				}
			}
			if allActed {
				room.PhaseDone <- true
			}
		} else {
			err = errors.New("当前不是狼人阶段")
		}

	case pb.Role_WITCH:
		if room.CurrentPhase == pb.Phase_PHASE_NIGHT_WITCH {
			if req.ActionType == "save" && !room.WitchSaveUsed {
				room.WitchSaveTarget = req.TargetPlayerId
				room.WitchSaveUsed = true
				result = "使用解药成功"
			} else if req.ActionType == "poison" && !room.WitchPoisonUsed {
				room.WitchPoisonTarget = req.TargetPlayerId
				room.WitchPoisonUsed = true
				result = "使用毒药成功"
			} else if req.ActionType == "skip" {
				result = "女巫不使用药水"
			} else {
				err = errors.New("药水已用过或操作无效")
			}

			if err == nil {
				player.CanAct = false
				room.PhaseDone <- true
			}
		} else {
			err = errors.New("当前不是女巫阶段")
		}

	case pb.Role_SEER:
		if room.CurrentPhase == pb.Phase_PHASE_NIGHT_SEER {
			target := room.Players[req.TargetPlayerId]
			if target.Camp == pb.Camp_CAMP_WEREWOLF {
				result = "这是一个狼人"
			} else {
				result = "这是一个好人"
			}
			player.CanAct = false
			room.PhaseDone <- true
		} else {
			err = errors.New("当前不是预言家阶段")
		}

	default:
		err = errors.New("无效的角色")
	}

	if err != nil {
		return &pb.NightActionResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	// 记录行动
	room.NightActions[req.PlayerId] = &pb.NightAction{
		PlayerId:   req.PlayerId,
		Role:       player.Role,
		TargetId:   req.TargetPlayerId,
		ActionType: req.ActionType,
		Timestamp:  time.Now().Unix(),
	}

	return &pb.NightActionResponse{
		Success: true,
		Message: "行动成功",
		Result:  result,
	}, nil
}

// Vote 投票
func (s *WerewolfServer) Vote(ctx context.Context, req *pb.VoteRequest) (*pb.VoteResponse, error) {
	s.mu.RLock()
	room, exists := s.rooms[req.RoomId]
	s.mu.RUnlock()

	if !exists {
		return nil, status.Error(codes.NotFound, "房间不存在")
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if room.CurrentPhase != pb.Phase_PHASE_DAY_VOTING {
		return &pb.VoteResponse{
			Success: false,
			Message: "当前不是投票阶段",
		}, nil
	}

	player := room.Players[req.VoterId]
	if !player.IsAlive {
		return &pb.VoteResponse{
			Success: false,
			Message: "死亡玩家不能投票",
		}, nil
	}

	room.Votes[req.VoterId] = req.TargetId

	// 检查是否所有人都投票了
	allVoted := true
	for _, p := range room.Players {
		if p.IsAlive {
			if _, voted := room.Votes[p.PlayerId]; !voted {
				allVoted = false
				break
			}
		}
	}

	if allVoted {
		room.PhaseDone <- true
	}

	return &pb.VoteResponse{
		Success: true,
		Message: "投票成功",
	}, nil
}

// GetGameState 获取游戏状态
func (s *WerewolfServer) GetGameState(ctx context.Context, req *pb.GetGameStateRequest) (*pb.GetGameStateResponse, error) {
	s.mu.RLock()
	room, exists := s.rooms[req.RoomId]
	s.mu.RUnlock()
	if !exists {
		return nil, status.Error(codes.NotFound, "房间不存在")
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	players := make([]*pb.Player, 0, len(room.Players))
	var currentPlayer *pb.Player

	for _, player := range room.Players {
		visiblePlayer := &pb.Player{
			PlayerId: player.PlayerId,
			Name:     player.Name,
			IsAlive:  player.IsAlive,
			Position: player.Position,
			CanAct:   player.CanAct,
		}

		// 只有玩家自己能看到自己的角色
		if player.PlayerId == req.PlayerId {
			visiblePlayer.Role = player.Role
			visiblePlayer.Camp = player.Camp
			currentPlayer = player
		} else {
			visiblePlayer.Role = pb.Role_UNKNOWN
			visiblePlayer.Camp = pb.Camp_CAMP_UNKNOWN
		}

		players = append(players, visiblePlayer)
	}

	return &pb.GetGameStateResponse{
		RoomId:        room.ID,
		State:         room.State,
		PhaseInfo:     room.getCurrentPhaseInfo(),
		Players:       players,
		DayCount:      int32(room.DayCount),
		CurrentPlayer: currentPlayer,
	}, nil
}

// SubscribeGameEvents 订阅游戏事件
func (s *WerewolfServer) SubscribeGameEvents(req *pb.GetGameStateRequest, stream pb.WerewolfService_SubscribeGameEventsServer) error {
	s.mu.RLock()
	room, exists := s.rooms[req.RoomId]
	s.mu.RUnlock()
	if !exists {
		return status.Error(codes.NotFound, "房间不存在")
	}

	// 创建事件通道
	eventChan := make(chan *pb.GameEvent, 100)

	room.mu.Lock()
	room.Subscribers[req.PlayerId] = eventChan
	room.mu.Unlock()

	// 清理订阅
	defer func() {
		room.mu.Lock()
		delete(room.Subscribers, req.PlayerId)
		close(eventChan)
		room.mu.Unlock()
	}()

	// 发送事件流
	for {
		select {
		case event := <-eventChan:
			if err := stream.Send(event); err != nil {
				return err
			}
		case <-stream.Context().Done():
			return nil
		}
	}
}

// 辅助方法
func (room *GameRoom) broadcastEvent(event *pb.GameEvent) {
	for _, ch := range room.Subscribers {
		select {
		case ch <- event:
		default:
			// 通道满了，跳过
		}
	}
}
func (room *GameRoom) getCurrentPhaseInfo() *pb.PhaseInfo {
	phaseNames := map[pb.Phase]string{
		pb.Phase_PHASE_WAITING:        "等待中",
		pb.Phase_PHASE_NIGHT_GUARD:    "守卫行动",
		pb.Phase_PHASE_NIGHT_WEREWOLF: "狼人行动",
		pb.Phase_PHASE_NIGHT_WITCH:    "女巫行动",
		pb.Phase_PHASE_NIGHT_SEER:     "预言家行动",
		pb.Phase_PHASE_DAY_DISCUSSION: "白天讨论",
		pb.Phase_PHASE_DAY_VOTING:     "投票",
		pb.Phase_PHASE_DAY_LAST_WORDS: "遗言",
		pb.Phase_PHASE_GAME_OVER:      "游戏结束",
	}
	return &pb.PhaseInfo{
		CurrentPhase: room.CurrentPhase,
		PhaseName:    phaseNames[room.CurrentPhase],
		TimeLimit:    60,
	}
}
func (room *GameRoom) nextPhase() {
	phaseOrder := []pb.Phase{
		pb.Phase_PHASE_NIGHT_GUARD,
		pb.Phase_PHASE_NIGHT_WEREWOLF,
		pb.Phase_PHASE_NIGHT_WITCH,
		pb.Phase_PHASE_NIGHT_SEER,
		pb.Phase_PHASE_DAY_DISCUSSION,
		pb.Phase_PHASE_DAY_VOTING,
		pb.Phase_PHASE_DAY_LAST_WORDS,
	}
	for i, phase := range phaseOrder {
		if room.CurrentPhase == phase {
			if i+1 < len(phaseOrder) {
				room.CurrentPhase = phaseOrder[i+1]
			} else {
				// 一天结束，进入下一个夜晚
				room.DayCount++
				room.State = pb.GameState_NIGHT
				room.CurrentPhase = pb.Phase_PHASE_NIGHT_GUARD

				// 重置夜晚数据
				room.NightActions = make(map[string]*pb.NightAction)
				room.GuardTarget = ""
				room.WerewolfTarget = ""
				room.WitchSaveTarget = ""
				room.WitchPoisonTarget = ""
			}

			if room.CurrentPhase == pb.Phase_PHASE_DAY_DISCUSSION {
				room.State = pb.GameState_DAY
			}

			return
		}
	}
}
func (room *GameRoom) settleNightDeaths() []*pb.Player {
	deadPlayers := make([]*pb.Player, 0)
	// 1. 判断狼人击杀
	if room.WerewolfTarget != "" {
		// 2. 判断守卫是否守护
		if room.WerewolfTarget != room.GuardTarget {
			// 3. 判断女巫是否救人
			if room.WitchSaveTarget != room.WerewolfTarget {
				// 玩家死亡
				player := room.Players[room.WerewolfTarget]
				player.IsAlive = false
				room.DeadPlayers[player.PlayerId] = true
				deadPlayers = append(deadPlayers, player)
			}
		}
	}

	// 4. 女巫毒人
	if room.WitchPoisonTarget != "" {
		player := room.Players[room.WitchPoisonTarget]
		if player.IsAlive {
			player.IsAlive = false
			room.DeadPlayers[player.PlayerId] = true
			deadPlayers = append(deadPlayers, player)
		}
	}

	return deadPlayers
}
func (room *GameRoom) countVotes() *pb.Player {
	voteCount := make(map[string]int)
	for _, targetID := range room.Votes {
		voteCount[targetID]++
	}

	maxVotes := 0
	var votedOutID string

	for playerID, count := range voteCount {
		if count > maxVotes {
			maxVotes = count
			votedOutID = playerID
		}
	}

	if votedOutID != "" {
		return room.Players[votedOutID]
	}

	return nil
}
func (room *GameRoom) checkGameOver() pb.Camp {
	werewolfCount := 0
	villagerCount := 0
	for _, player := range room.Players {
		if player.IsAlive {
			if player.Camp == pb.Camp_CAMP_WEREWOLF {
				werewolfCount++
			} else {
				villagerCount++
			}
		}
	}

	// 狼人全灭，好人胜利
	if werewolfCount == 0 {
		return pb.Camp_CAMP_VILLAGER
	}

	// 狼人数 >= 好人数，狼人胜利
	if werewolfCount >= villagerCount {
		return pb.Camp_CAMP_WEREWOLF
	}

	return pb.Camp_CAMP_UNKNOWN
}
func assignRoles(room *GameRoom) error {
	playerList := make([]*pb.Player, 0, len(room.Players))
	for _, player := range room.Players {
		playerList = append(playerList, player)
	}
	roles := make([]pb.Role, 0)
	for roleStr, count := range room.RoleConfig {
		role := stringToRole(roleStr)
		for i := 0; i < int(count); i++ {
			roles = append(roles, role)
		}
	}

	if len(roles) != len(playerList) {
		return errors.New("角色数量与玩家数量不匹配")
	}

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(roles), func(i, j int) {
		roles[i], roles[j] = roles[j], roles[i]
	})

	for i, player := range playerList {
		player.Role = roles[i]

		// 设置阵营
		if player.Role == pb.Role_WEREWOLF {
			player.Camp = pb.Camp_CAMP_WEREWOLF
		} else {
			player.Camp = pb.Camp_CAMP_VILLAGER
		}
	}

	return nil
}
func stringToRole(s string) pb.Role {
	roleMap := map[string]pb.Role{
		"werewolf": pb.Role_WEREWOLF,
		"villager": pb.Role_VILLAGER,
		"seer":     pb.Role_SEER,
		"witch":    pb.Role_WITCH,
		"hunter":   pb.Role_HUNTER,
		"guard":    pb.Role_GUARD,
	}
	if role, ok := roleMap[s]; ok {
		return role
	}
	return pb.Role_UNKNOWN
}
func getCampName(camp pb.Camp) string {
	if camp == pb.Camp_CAMP_WEREWOLF {
		return "狼人"
	}
	return "好人"
}
func generateRoomID() string {
	return fmt.Sprintf("room_%d", time.Now().UnixNano())
}
func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
