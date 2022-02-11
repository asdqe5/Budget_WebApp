// 프로젝트 결산 프로그램
//
// Description : 유저 자료구조 스크립트

package main

import (
	"errors"

	"github.com/dgrijalva/jwt-go"
)

// Token 자료구조. JWT 방식을 사용한다. restAPI 사용시 보안체크를 위해 http 헤더에 들어간다.
type Token struct {
	ID          string      `json:"id" bson:"id"`                   // 사용자 ID
	AccessLevel AccessLevel `json:"accesslevel" bson:"accesslevel"` // 액세스 레벨
	ToolName    string      `json:"toolname" bson:"toolname"`       // 툴 이름
	jwt.StandardClaims
}

// AccessLevel 유저의 액세스 레벨
type AccessLevel int

const (
	GuestLevel = AccessLevel(iota)
	DefaultLevel
	MemberLevel
	ManagerLevel
	AdminLevel
)

// User 사용자 자료구조이다.
type User struct {
	ID          string      `json:"id" bson:"id"`                   // 사용자 ID
	Password    string      `json:"password" bson:"password"`       // 암호화된 비밀번호
	Name        string      `json:"name" bson:"name"`               // 사용자 이름
	Team        string      `json:"team" bson:"team"`               // 사용자 팀
	Token       string      `json:"token" bson:"token"`             // JWT 토큰
	SignKey     string      `json:"signkey" bson:"signkey"`         // JWT 토큰을 만들 때 사용하는 SignKey
	AccessLevel AccessLevel `json:"accesslevel" bson:"accesslevel"` // 액세스 레벨
}

// CheckError 함수는 User 자료구조에 값이 정확히 들어갔는지 확인하는 함수이다.
func (u User) CheckError() error {
	if u.ID == "" {
		return errors.New("ID를 입력해주세요")
	}
	if u.Password == "" {
		return errors.New("password를 입력해주세요")
	}
	if u.Name == "" {
		return errors.New("name을 입력해주세요")
	}
	if !regexName.MatchString(u.Name) {
		return errors.New("name에는 한글, 영문만 사용가능합니다")
	}
	if u.Team == "" {
		return errors.New("team을 입력해주세요")
	}
	return nil
}

// CreateToken 메소드는 토큰을 생성합니다.
func (u *User) CreateToken() error {
	if u.ID == "" {
		return errors.New("ID is an empty string")
	}
	if u.Password == "" {
		return errors.New("Password is an empty string")
	}
	token := jwt.NewWithClaims(jwt.GetSigningMethod("HS256"), &Token{
		ID:          u.ID,
		AccessLevel: u.AccessLevel,
		ToolName:    "budget",
	})
	signKey, err := encryptFunc(u.Password)
	if err != nil {
		return err
	}
	u.SignKey = signKey
	tokenString, err := token.SignedString([]byte(signKey))
	if err != nil {
		return err
	}
	u.Token = tokenString
	return nil
}

func (al AccessLevel) String() string {
	switch al {
	case 0:
		return "guest"
	case 1:
		return "default"
	case 2:
		return "member"
	case 3:
		return "manager"
	case 4:
		return "admin"
	}
	return ""
}
