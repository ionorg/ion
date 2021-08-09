package server

import (
	"context"

	room "github.com/pion/ion/apps/room/proto"
)

/*
string sid = 1;
string uid = 2;
string displayName = 3;
bytes extraInfo = 4;
Role role = 5;
string avatar = 6;
string vendor = 7;
string token = 8;
*/

type participant struct {
	uid         string
	displayName string
	room        *Room
	sig         room.Room_SignalServer
	ctx         context.Context
}

func (p *participant) Close() {

}
