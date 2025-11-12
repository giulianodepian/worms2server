package serverhandler

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"sync"
)

type ActionCode uint32

const (
	ListRooms       ActionCode = 200
	ListUsers       ActionCode = 400
	ListGames       ActionCode = 500
	LoginQuery      ActionCode = 600
	CreateRoom      ActionCode = 700
	JoinRoomOrGame  ActionCode = 800
	LeaveRoomOrGame ActionCode = 900
	CloseRoomOrGame ActionCode = 1100
	CreateGame      ActionCode = 1200
)

type SessionInfo struct {
	magicNumber1 uint32
	magicNumber2 uint32
	flag         uint8
	gameVer      uint8
	gameRelease  uint8
	sessionType  uint8
	access       uint8
	magicNumber3 uint8
	magicNumber4 uint8
	padding      []byte
}

type Packet struct {
	code        uint32
	flags       [11]bool
	value0      uint32
	value1      uint32
	value2      uint32
	value3      uint32
	value4      uint32
	value10     uint32
	dataLen     uint32
	data        []byte
	error       uint32
	name        []byte
	sessionInfo SessionInfo
}

type RoomInfo struct {
	creatorIp   []byte
	name        string
	sessionInfo SessionInfo
	userIds     []uint32
	gamesIds    []uint32
}

type UserInfo struct {
	name        string
	ipAddress   string
	sessionInfo SessionInfo
	//conn net.Conn //We will need this for Notice type of messages
}

type GameInfo struct {
	name        string
	hostIp      []byte
	sessionInfo SessionInfo
}

var idCounter uint32 = 0x1000

var mu sync.Mutex

var connectedUsers = make(map[uint32]UserInfo)
var rooms = make(map[uint32]RoomInfo)
var games = make(map[uint32]GameInfo)

func littleToBigEndianDecode(packetData []byte, start uint32) uint32 {
	return (uint32(packetData[start+3]) << 24) + (uint32(packetData[start+2]) << 16) + (uint32(packetData[start+1]) << 8) + uint32(packetData[start])
}

func bigEndianToLittleEndianEncode(value uint32) []byte {
	encodedValue := make([]byte, 4)
	binary.LittleEndian.PutUint32(encodedValue, value)
	return encodedValue
}

func BytesToPacket(packetData []byte) *Packet {
	fmt.Println("Received message: ", hex.EncodeToString(packetData))
	packet := Packet{}
	packet.code = littleToBigEndianDecode(packetData, 0)
	flags := littleToBigEndianDecode(packetData, 4)
	fmt.Println("Action Code: ", packet.code)
	fmt.Println("Flags = ", flags)
	var offset uint32 = 8
	isFlag10Set := flags&(1<<10) == (1 << 10)
	var value10offset uint32 = 0
	for i := range 11 {
		packet.flags[i] = flags&(1<<i) == (1 << i)
		if packet.flags[i] {
			switch i {
			case 0:
				{
					packet.value0 = littleToBigEndianDecode(packetData, offset)
					fmt.Println("Value 0 = ", packet.value0)
					offset += 4
				}
			case 1:
				{
					packet.value1 = littleToBigEndianDecode(packetData, offset)
					fmt.Println("Value 1 = ", packet.value1)
					offset += 4
				}
			case 2:
				{
					packet.value2 = littleToBigEndianDecode(packetData, offset)
					fmt.Println("Value 2 = ", packet.value2)
					offset += 4
				}
			case 3:
				{
					packet.value3 = littleToBigEndianDecode(packetData, offset)
					fmt.Println("Value 3 = ", packet.value3)
					offset += 4
				}
			case 4:
				{
					packet.value4 = littleToBigEndianDecode(packetData, offset)
					fmt.Println("Value 4 = ", packet.value4)
					offset += 4
				}
			case 5:
				{
					if isFlag10Set {
						value10offset = offset
						offset += 4
						isFlag10Set = false
					}
					packet.dataLen = littleToBigEndianDecode(packetData, offset)
					fmt.Println("Datalen = ", packet.dataLen)
					offset += 4
				}
			case 6:
				{
					if isFlag10Set {
						value10offset = offset
						offset += 4
						isFlag10Set = false
					}
					packet.data = packetData[offset : offset+packet.dataLen]
					fmt.Println("Data = ", packet.data)
					offset += packet.dataLen
				}
			case 7:
				{
					if isFlag10Set {
						value10offset = offset
						offset += 4
						isFlag10Set = false
					}
					packet.error = littleToBigEndianDecode(packetData, offset)
					fmt.Println("error = ", packet.error)
					offset += 4
				}
			case 8:
				{
					if isFlag10Set {
						value10offset = offset
						offset += 4
						isFlag10Set = false
					}
					packet.name = packetData[offset : offset+20]
					fmt.Println("name = ", packet.name)
					offset += 20
				}
			case 9:
				{
					if isFlag10Set {
						value10offset = offset
						offset += 4
						isFlag10Set = false
					}
					packet.sessionInfo = SessionInfo{
						magicNumber1: littleToBigEndianDecode(packetData, offset),
						magicNumber2: littleToBigEndianDecode(packetData, offset+4),
						flag:         packetData[offset+8],
						gameVer:      packetData[offset+9],
						gameRelease:  packetData[offset+10],
						sessionType:  packetData[offset+11],
						access:       packetData[offset+12],
						magicNumber3: packetData[offset+13],
						magicNumber4: packetData[offset+14],
						padding:      packetData[offset+15 : offset+15+35],
					}
					fmt.Println("Session Data: ", packet.sessionInfo)
					offset += 50
				}
			case 10:
				{
					if isFlag10Set {
						value10offset = offset
						isFlag10Set = false
					}
					packet.value10 = littleToBigEndianDecode(packetData, value10offset)
					fmt.Println("value 10 = ", packet.value10)
				}
			}
		}
	}
	return &packet
}

func (p *Packet) ToBytes() []byte {
	var bytesToSend []byte
	bytesToSend = append(bytesToSend, bigEndianToLittleEndianEncode(p.code)...)
	var intFlag uint32 = 0
	for i := range 11 {
		if p.flags[i] {
			intFlag += 1 << i
		}
	}
	bytesToSend = append(bytesToSend, bigEndianToLittleEndianEncode(intFlag)...)
	for i := range 10 {
		if i == 5 && p.flags[10] {
			bytesToSend = append(bytesToSend, bigEndianToLittleEndianEncode(p.value10)...)
		}
		if p.flags[i] {
			switch i {
			case 0:
				{
					bytesToSend = append(bytesToSend, bigEndianToLittleEndianEncode(p.value0)...)
				}
			case 1:
				{
					bytesToSend = append(bytesToSend, bigEndianToLittleEndianEncode(p.value1)...)
				}
			case 2:
				{
					bytesToSend = append(bytesToSend, bigEndianToLittleEndianEncode(p.value2)...)
				}
			case 3:
				{
					bytesToSend = append(bytesToSend, bigEndianToLittleEndianEncode(p.value3)...)
				}
			case 4:
				{
					bytesToSend = append(bytesToSend, bigEndianToLittleEndianEncode(p.value4)...)
				}
			case 5:
				{
					bytesToSend = append(bytesToSend, bigEndianToLittleEndianEncode(p.dataLen)...)
				}
			case 6:
				{
					bytesToSend = append(bytesToSend, p.data...)
				}
			case 7:
				{
					bytesToSend = append(bytesToSend, bigEndianToLittleEndianEncode(p.error)...)
				}
			case 8:
				{
					bytesToSend = append(bytesToSend, p.name...)
				}
			case 9:
				{
					bytesToSend = append(bytesToSend, bigEndianToLittleEndianEncode(p.sessionInfo.magicNumber1)...)
					bytesToSend = append(bytesToSend, bigEndianToLittleEndianEncode(p.sessionInfo.magicNumber2)...)
					bytesToSend = append(bytesToSend, p.sessionInfo.flag)
					bytesToSend = append(bytesToSend, p.sessionInfo.gameVer)
					bytesToSend = append(bytesToSend, p.sessionInfo.gameRelease)
					bytesToSend = append(bytesToSend, p.sessionInfo.sessionType)
					bytesToSend = append(bytesToSend, p.sessionInfo.access)
					bytesToSend = append(bytesToSend, p.sessionInfo.magicNumber3)
					bytesToSend = append(bytesToSend, p.sessionInfo.magicNumber4)
					bytesToSend = append(bytesToSend, p.sessionInfo.padding...)
				}
			}
		}
	}
	return bytesToSend
}

func HandleClientData(conn net.Conn, clientData []byte, ipAddress string) {
	packet := BytesToPacket(clientData)
	fmt.Println("----------------------------------------------------------------------------")
	switch packet.code {
	case uint32(LoginQuery):
		{
			mu.Lock()
			connectedUsers[idCounter] = UserInfo{
				name:        string(packet.name),
				ipAddress:   ipAddress,
				sessionInfo: packet.sessionInfo,
			}
			packetToSent := Packet{
				code:   601,
				flags:  [11]bool{false, true, false, false, false, false, false, false, false, false, false},
				value1: idCounter,
			}
			idCounter++
			mu.Unlock()
			conn.Write(packetToSent.ToBytes())
		}
	case uint32(ListRooms):
		{
			for id, info := range rooms {
				packetToSent := Packet{
					code:        350,
					flags:       [11]bool{false, true, false, false, false, true, true, true, true, true, false},
					value1:      id,
					dataLen:     (uint32)(len(info.creatorIp)),
					data:        info.creatorIp,
					error:       0,
					name:        []byte(info.name),
					sessionInfo: info.sessionInfo,
				}
				conn.Write(packetToSent.ToBytes())
			}
			packetToSent := Packet{
				code:  351,
				flags: [11]bool{false},
			}
			conn.Write(packetToSent.ToBytes())
		}
	case uint32(CreateRoom):
		{
			mu.Lock()
			rooms[idCounter] = RoomInfo{
				creatorIp:   packet.data,
				name:        string(packet.name),
				sessionInfo: packet.sessionInfo,
			}
			packetToSent := Packet{
				code:   701,
				flags:  [11]bool{false, true, false, false, false, false, false, false, false, false},
				value1: idCounter,
			}
			idCounter++
			mu.Unlock()
			conn.Write(packetToSent.ToBytes())
		}
	case uint32(JoinRoomOrGame):
		{
			room := rooms[packet.value2]
			room.userIds = append(room.userIds, packet.value10)
			rooms[packet.value2] = room
			packetToSent := Packet{
				code:  801,
				flags: [11]bool{false},
			}
			conn.Write(packetToSent.ToBytes())
		}
	case uint32(ListGames):
		{
			for _, id := range rooms[packet.value2].gamesIds {
				game := games[id]
				packetToSent := Packet{
					code:        350,
					flags:       [11]bool{false, true, false, false, false, true, true, true, true, true, false},
					value1:      uint32(id),
					dataLen:     uint32(len(game.hostIp)),
					data:        game.hostIp,
					error:       0,
					name:        []byte(game.name),
					sessionInfo: game.sessionInfo,
				}
				conn.Write(packetToSent.ToBytes())
			}
			packetToSent := Packet{
				code:  351,
				flags: [11]bool{false, false, false, false, false, false, false, true, false, false, false},
				error: 0,
			}
			conn.Write(packetToSent.ToBytes())
		}
	case uint32(ListUsers):
		{

			for _, id := range rooms[packet.value2].userIds {
				user := connectedUsers[id]
				packetToSent := Packet{
					code:        350,
					flags:       [11]bool{false, true, false, false, false, true, true, true, true, true, false},
					value1:      uint32(id),
					dataLen:     uint32(len(user.ipAddress) + 1),
					data:        append([]byte(user.ipAddress), 0),
					error:       0,
					name:        []byte(user.name),
					sessionInfo: user.sessionInfo,
				}
				conn.Write(packetToSent.ToBytes())
			}

			packetToSent := Packet{
				code:  351,
				flags: [11]bool{false, false, false, false, false, false, false, true, false, false, false},
				error: 0,
			}
			conn.Write(packetToSent.ToBytes())
		}
	case uint32(LeaveRoomOrGame):
		{
			var userToRemove = -1
			for index, id := range rooms[packet.value2].userIds {
				if id == packet.value10 {
					userToRemove = index
				}
			}
			if userToRemove != -1 {
				room := rooms[packet.value2]
				room.userIds = append(room.userIds[:userToRemove], room.userIds[userToRemove+1:]...)
				rooms[packet.value2] = room
			}
			packetToSent := Packet{
				code:  901,
				flags: [11]bool{false, false, false, false, false, false, false, true, false, false, false},
				error: 0,
			}
			conn.Write(packetToSent.ToBytes())
		}
	case uint32(CloseRoomOrGame):
		{
			delete(rooms, packet.value10)
			packetToSent := Packet{
				code:  1101,
				flags: [11]bool{false, false, false, false, false, false, false, true, false, false, false},
				error: 0,
			}
			conn.Write(packetToSent.ToBytes())
		}
	case uint32(CreateGame):
		{
			//To be fully functional, we have to combine rooms and games in the same list, since they are treated the same, and identify them with a type
			mu.Lock()
			games[idCounter] = GameInfo{
				hostIp:      packet.data,
				name:        string(packet.name),
				sessionInfo: packet.sessionInfo,
			}
			packetToSent := Packet{
				code:   1201,
				flags:  [11]bool{false, true, false, false, false, false, false, false, false, false},
				value1: idCounter,
			}
			idCounter++
			mu.Unlock()
			conn.Write(packetToSent.ToBytes())
		}
	}
}

func HandleDisconnection(ipAddress string) {
	fmt.Println("ipAddress Disconnected: ", ipAddress)
}
