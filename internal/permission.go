package internal

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type PermissionLevel int

const (
	PermOwner PermissionLevel = iota
	PermGroup
	PermEveryone
)

func GetPermission(col string) (owner string, group string, everyone string) {
	// default permission
	owner, group, everyone = "7", "4", "0"

	re := regexp.MustCompile(`_\d\d\d_$`)
	if !re.MatchString(col) {
		return
	}

	results := re.FindAllString(col, -1)
	if len(results) != 1 {
		return
	}

	perm := strings.Replace(results[0], "_", "", -1)

	if len(perm) != 3 {
		return
	}

	owner = string(perm[0])
	group = string(perm[1])
	everyone = string(perm[2])
	return
}

func WritePermission(col string) PermissionLevel {
	_, g, e := GetPermission(col)

	if CanWrite(e) {
		return PermEveryone
	}
	if CanWrite(g) {
		return PermGroup
	}
	return PermOwner
}

func ReadPermission(col string) PermissionLevel {
	_, g, e := GetPermission(col)

	if CanRead(e) {
		return PermEveryone
	}
	if CanRead(g) {
		return PermGroup
	}
	return PermOwner
}

func CanWrite(s string) bool {
	i, err := strconv.Atoi(s)
	if err != nil {
		return false
	}
	return uint8(i)&uint8(2) != 0
}

func CanRead(s string) bool {
	i, err := strconv.Atoi(s)
	if err != nil {
		fmt.Println(err)
	}
	return uint8(i)&uint8(4) != 0
}
