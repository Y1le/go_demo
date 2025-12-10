// controllers/user/user_controller_test.go
package user

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"liam/dto"
	"liam/mocks"
	"liam/pkg/errors"
	"liam/utils"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUserController_Login_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUserService := new(mocks.UserService)
	mockEmailService := new(mocks.EmailService)

	userReq := dto.LoginRequest{Email: "test@example.com", Password: "secure123"}
	userToken, err := utils.GenerateToken(1, "小红")
	assert.NoError(t, err)
	userResp := &dto.UserResponse{ID: 1, Email: "test@example.com", Token: userToken, Name: "小红"}
	t.Logf("userReq: %v", userReq)
	// mockUserService.On("Login", mock.Anything, &userReq).Return(userResp, nil)
	mockUserService.On("Login",
		mock.Anything,      // ctx context.Context
		"test@example.com", // email string
		"secure123",        // password string
	).Return(userResp, nil)
	controller := NewUserController(mockUserService, mockEmailService)

	routes := gin.New()
	routes.POST("/login", controller.Login)

	reqBody, _ := json.Marshal(userReq)
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(reqBody))
	w := httptest.NewRecorder()

	routes.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	t.Logf("resp: %v", resp["data"])
	var data = resp["data"].(map[string]interface{})
	assert.Equal(t, float64(1), data["id"])
	assert.Equal(t, "test@example.com", data["email"])
	assert.Equal(t, "小红", data["name"])

	mockUserService.AssertExpectations(t)
}

func TestUserController_GetUserByID_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUserService := new(mocks.UserService)
	mockEmailService := new(mocks.EmailService)

	userResp := &dto.UserResponse{ID: 1, Email: "test@example.com"}
	mockUserService.On("GetUserByID", mock.Anything, uint(1)).Return(userResp, nil)

	controller := NewUserController(mockUserService, mockEmailService)

	router := gin.New()
	router.GET("/users/:id", controller.GetUserByID)

	req := httptest.NewRequest("GET", "/users/1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	t.Logf("resp: %v", resp)
	assert.Equal(t, float64(1), resp["id"])
	assert.Equal(t, "test@example.com", resp["email"])

	mockUserService.AssertExpectations(t)
}

func TestUserController_GetUserByID_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUserService := new(mocks.UserService)
	mockEmailService := new(mocks.EmailService)
	controller := NewUserController(mockUserService, mockEmailService)

	router := gin.New()
	router.GET("/users/:id", controller.GetUserByID)

	req := httptest.NewRequest("GET", "/users/abc", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["message"], "Invalid user ID format")
}

func TestUserController_CreateUser_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUserService := new(mocks.UserService)
	mockEmailService := new(mocks.EmailService)

	reqBody := dto.CreateUserRequest{Email: "new@example.com", Name: "New User", Password: "secure123"}
	userResp := &dto.UserResponse{
		ID:    1,
		Email: "test@example.com",
		Name:  "John Doe",
	}
	mockUserService.On("CreateUser", mock.Anything, &reqBody).Return(userResp, nil)

	controller := NewUserController(mockUserService, mockEmailService)

	router := gin.New()
	router.POST("/users", controller.CreateUser)

	jsonData, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/users", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	t.Logf("w.Body: %s", w.Body.String())
	assert.Equal(t, http.StatusCreated, w.Code)
	mockUserService.AssertExpectations(t)
}

func TestUserController_CreateUser_BadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUserService := new(mocks.UserService)
	mockEmailService := new(mocks.EmailService)
	controller := NewUserController(mockUserService, mockEmailService)

	router := gin.New()
	router.POST("/users", controller.CreateUser)

	// Invalid JSON
	req := httptest.NewRequest("POST", "/users", bytes.NewBufferString(`{invalid}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(errors.ErrInvalidInput.Code), resp["code"])
}
