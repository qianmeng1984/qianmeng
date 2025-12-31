package service

import (
	"errors"
	"golang.org/x/crypto/bcrypt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
	"rag-knowledge-base/internal/model"
)

// 生产环境千万别用明文存密码，但毕设为了演示方便，我懂的！😉
var SecretKey = []byte("rune_student_secret_key")

type AuthService struct {
	db *gorm.DB
}

func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{db: db}
}

// Register 注册 (默认都是普通用户 Role=0)
func (s *AuthService) Register(username, password string) error {
	var count int64
	s.db.Model(&model.User{}).Where("username = ?", username).Count(&count)
	if count > 0 {
		return errors.New("用户名已存在")
	}

	hashedPwd, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	user := model.User{
		Username: username,
		Password: string(hashedPwd),
		Role:     0, // 默认普通用户
		Nickname: "新用户" + username,
		Avatar:   "/uploads/avatars/default.png", // 请确保你在 web/uploads/avatars/ 下放了一张 default.png
	}

	return s.db.Create(&user).Error
}

// Login 登录
func (s *AuthService) Login(username, password string) (string, uint, string, error) {
	var user model.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		return "", 0, "", errors.New("用户不存在")
	}

	// 加密对比
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", 0, "", errors.New("密码错误")
	}

	// 生成 JWT Token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString(SecretKey)

	roleName := "user"
	if user.Role == 1 {
		roleName = "admin"
	}

	return tokenString, user.ID, roleName, err
}

// 2. 新增：修改用户信息 (改昵称、头像、密码)
// 如果参数为空字符串，代表不修改该项
func (s *AuthService) UpdateUserInfo(userID uint, nickname, avatar, newPassword string) error {
	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return errors.New("用户不存在")
	}

	// 更新昵称
	if nickname != "" {
		user.Nickname = nickname
	}

	// 更新头像路径
	if avatar != "" {
		user.Avatar = avatar
	}

	// 更新密码 (如果用户填了新密码)
	if newPassword != "" {
		hashedPwd, _ := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		user.Password = string(hashedPwd)
	}

	return s.db.Save(&user).Error
}

// 3. 新增：获取当前用户信息 (用于前端回显)
func (s *AuthService) GetUserInfo(userID uint) (*model.User, error) {
	var user model.User
	// 注意：查出来后要把密码清空，不能传给前端
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}
	user.Password = ""
	return &user, nil
}

// GetAllUsers (管理员) 获取所有用户列表
func (s *AuthService) GetAllUsers() ([]model.User, error) {
	var users []model.User
	// 排除密码字段，不返回给前端
	err := s.db.Select("id, username, nickname, role, avatar, created_at").Find(&users).Error
	return users, err
}

// DeleteUser (管理员) 注销用户
func (s *AuthService) DeleteUser(targetUserID uint) error {
	// 硬删除或软删除均可，这里演示软删除
	return s.db.Delete(&model.User{}, targetUserID).Error
}

// AdminUpdateUser (管理员) 修改指定用户信息
func (s *AuthService) AdminUpdateUser(targetUserID uint, nickname string, password string) error {
	updates := map[string]interface{}{
		"nickname": nickname,
	}
	// 如果管理员输入了新密码，则重置密码
	if password != "" {
		hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		updates["password"] = string(hash)
	}
	return s.db.Model(&model.User{}).Where("id = ?", targetUserID).Updates(updates).Error
}
