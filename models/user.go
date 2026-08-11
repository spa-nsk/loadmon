package models

import (
	"errors"
	"strconv"
	"time"
)

var (
	// UserList — демонстрационное in-memory хранилище пользователей (пример из шаблона beego).
	UserList map[string]*User
)

func init() {
	UserList = make(map[string]*User)
	u := User{"user_11111", "astaxie", "11111", Profile{"male", 20, "Singapore", "astaxie@gmail.com"}}
	UserList["user_11111"] = &u
}

// User — демонстрационная структура пользователя (пример из шаблона beego), не связана с БД.
type User struct {
	Id       string
	Username string
	Password string
	Profile  Profile
}

// Profile — демонстрационные профильные данные пользователя (пример из шаблона beego).
type Profile struct {
	Gender  string
	Age     int
	Address string
	Email   string
}

// AddUser добавляет пользователя в хранилище UserList и возвращает сгенерированный Id.
func AddUser(u User) string {
	u.Id = "user_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	UserList[u.Id] = &u
	return u.Id
}

// GetUser возвращает пользователя из хранилища UserList по идентификатору.
func GetUser(uid string) (u *User, err error) {
	if u, ok := UserList[uid]; ok {
		return u, nil
	}
	return nil, errors.New("User not exists")
}

// GetAllUsers возвращает всех пользователей из хранилища UserList.
func GetAllUsers() map[string]*User {
	return UserList
}

// UpdateUser обновляет непустые поля пользователя uid данными из uu.
func UpdateUser(uid string, uu *User) (a *User, err error) {
	if u, ok := UserList[uid]; ok {
		if uu.Username != "" {
			u.Username = uu.Username
		}
		if uu.Password != "" {
			u.Password = uu.Password
		}
		if uu.Profile.Age != 0 {
			u.Profile.Age = uu.Profile.Age
		}
		if uu.Profile.Address != "" {
			u.Profile.Address = uu.Profile.Address
		}
		if uu.Profile.Gender != "" {
			u.Profile.Gender = uu.Profile.Gender
		}
		if uu.Profile.Email != "" {
			u.Profile.Email = uu.Profile.Email
		}
		return u, nil
	}
	return nil, errors.New("User Not Exist")
}

// Login проверяет наличие пользователя с указанными логином и паролем в UserList.
func Login(username, password string) bool {
	for _, u := range UserList {
		if u.Username == username && u.Password == password {
			return true
		}
	}
	return false
}

// DeleteUser удаляет пользователя из хранилища UserList по идентификатору.
func DeleteUser(uid string) {
	delete(UserList, uid)
}
