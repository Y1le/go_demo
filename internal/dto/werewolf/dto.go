package dto

// 请求 DTO
type CreateRoomRequest struct {
	RoomName   string         `json:"room_name" binding:"required"`
	MaxPlayers int            `json:"max_players" binding:"required,min=4,max=12"`
	RoleConfig map[string]int `json:"role_config" binding:"required"`
}

type JoinRoomRequest struct {
	RoomID     string `json:"room_id" binding:"required"`
	PlayerID   string `json:"player_id,omitempty"` // 由服务器生成
	PlayerName string `json:"player_name" binding:"required,min=1,max=20"`
}

type StartGameRequest struct {
	RoomID string `json:"room_id" binding:"required"`
}

type NightActionRequest struct {
	RoomID     string `json:"room_id" binding:"required"`
	PlayerID   string `json:"player_id" binding:"required"`
	TargetID   string `json:"target_id" binding:"required"`
	ActionType string `json:"action_type" binding:"required"` // kill, check, save, poison, guard, skip
}

type VoteRequest struct {
	RoomID   string `json:"room_id" binding:"required"`
	VoterID  string `json:"voter_id" binding:"required"`
	TargetID string `json:"target_id" binding:"required"`
}

type LeaveRoomRequest struct {
	RoomID   string `json:"room_id" binding:"required"`
	PlayerID string `json:"player_id" binding:"required"`
}

// 响应 DTO
type CreateRoomResponse struct {
	RoomID  string `json:"room_id"`
	Message string `json:"message"`
}

type JoinRoomResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	PlayerID string `json:"player_id,omitempty"`
	Role     string `json:"role,omitempty"`
	Position int32  `json:"position,omitempty"`
}

type StartGameResponse struct {
	Success   bool       `json:"success"`
	Message   string     `json:"message"`
	PhaseInfo *PhaseInfo `json:"phase_info,omitempty"`
}

type NightActionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Result  string `json:"result,omitempty"`
}

type VoteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type GetGameStateResponse struct {
	RoomID        string       `json:"room_id"`
	State         string       `json:"state"`
	PhaseInfo     *PhaseInfo   `json:"phase_info"`
	Players       []PlayerInfo `json:"players"`
	DayCount      int          `json:"day_count"`
	CurrentPlayer *PlayerInfo  `json:"current_player,omitempty"`
}

type RoomPlayersResponse struct {
	Success     bool         `json:"success"`
	RoomID      string       `json:"room_id"`
	Players     []PlayerInfo `json:"players"`
	OnlineCount int          `json:"online_count"`
}

// 通用结构
type PlayerInfo struct {
	PlayerID string `json:"player_id"`
	Name     string `json:"name"`
	Role     string `json:"role,omitempty"`
	Camp     string `json:"camp,omitempty"`
	IsAlive  bool   `json:"is_alive"`
	Position int32  `json:"position"`
	CanAct   bool   `json:"can_act"`
}

type PhaseInfo struct {
	CurrentPhase string   `json:"current_phase"`
	PhaseName    string   `json:"phase_name"`
	ActiveRoles  []string `json:"active_roles,omitempty"`
	TimeLimit    int32    `json:"time_limit"`
	Description  string   `json:"description,omitempty"`
}

// 通用响应
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Message string `json:"message"`
}
