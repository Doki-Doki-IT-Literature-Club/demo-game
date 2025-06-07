package types

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
)

type ObjectID uint32

type Command byte

const (
	UP    = 0x01
	DOWN  = 0x02
	LEFT  = 0x03
	RIGHT = 0x04
	SHOOT = 0x05
)

type Direction uint32

const (
	D_UP = iota
	D_DOWN
	D_LEFT
	D_RIGHT
)

func (d Direction) AsVector() Vector {
	switch d {
	case D_UP:
		return Vector{0, 1}
	case D_DOWN:
		return Vector{0, -1}
	case D_LEFT:
		return Vector{-1, 0}
	case D_RIGHT:
		return Vector{1, 0}
	}
	panic("Unsupported direction")
}

func (d Direction) AsRune() rune {
	switch d {
	case D_UP:
		return '^'
	case D_DOWN:
		return 'V'
	case D_LEFT:
		return '<'
	case D_RIGHT:
		return '>'
	}
	panic("Unsupported direction")
}

const FieldMaxX = 500
const FieldMaxY = 500

type Vector struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func (v Vector) Add(other Vector) Vector {
	return Vector{v.X + other.X, v.Y + other.Y}
}

func (v Vector) Sub(other Vector) Vector {
	return Vector{v.X - other.X, v.Y - other.Y}
}

func (v Vector) SingleVector() Vector {
	vectorLen := v.GetLen()
	if math.IsNaN(vectorLen) {
		return Vector{}
	}
	return Vector{v.X / vectorLen, v.Y / vectorLen}
}

func (v Vector) GetLen() float64 {
	return math.Sqrt(math.Pow(v.X, 2) + math.Pow(v.Y, 2))
}

func (v Vector) Multiply(a float64) Vector {
	return Vector{v.X * a, v.Y * a}
}

func (v Vector) ToString() string {
	return fmt.Sprintf("{X: %.2f | Y: %.2f}", v.X, v.Y)
}

type CollisionArea Vector

func (ca CollisionArea) ToCollisionBox(position Vector) CollisionBox {
	return CollisionBox{
		BottomLeft: position,
		TopRight:   position.Add(Vector(ca)),
		IsRigid:    true,
	}
}

type CollidableObject interface {
	GetPosition() Vector
	GetCollisionArea() CollisionArea
	OnCollision(CollidableObject)
}

type MovableObject interface {
	CollidableObject

	GetID() ObjectID
	GetSpeed() Vector
	SetSpeed(Vector)

	SetPosition(Vector)
}

type Player struct {
	ID            ObjectID
	Position      Vector
	CollisionArea CollisionArea
	Speed         Vector
	IsAirborn     bool
	ViewDirection Direction
	HP            uint32
}

func (p *Player) ToString() string {
	airbornStr := "[S]"
	if p.IsAirborn {
		airbornStr = "[A]"
	}
	return fmt.Sprintf("Player: %c%s, Position: %s, Speed: %s", p.ViewDirection.AsRune(), airbornStr, p.Position.ToString(), p.Speed.ToString())
}

func (p Player) GetID() ObjectID {
	return p.ID
}

func (p Player) GetSpeed() Vector {
	return p.Speed
}

func (p *Player) SetSpeed(speed Vector) {
	p.Speed = speed
}

func (p Player) GetPosition() Vector {
	return p.Position
}

func (p *Player) SetPosition(position Vector) {
	p.Position = position
}

func (p Player) GetCollisionArea() CollisionArea {
	return p.CollisionArea
}

func (p Player) GetCollisionBox() CollisionBox {
	return p.CollisionArea.ToCollisionBox(p.Position)
}

func (p *Player) OnCollision(co CollidableObject) {
	switch co.(type) {
	case *Projectile:
		p.HP -= 1
	default:
		return
	}
}

func (p Player) ToBytes() []byte {
	pb := [20]byte{}
	binary.BigEndian.PutUint32(pb[:4], uint32(p.ID))
	binary.BigEndian.PutUint32(pb[4:8], uint32(p.ViewDirection))
	binary.BigEndian.PutUint32(pb[8:12], uint32(math.Round(p.Position.X)))
	binary.BigEndian.PutUint32(pb[12:16], uint32(math.Round(p.Position.Y)))
	binary.BigEndian.PutUint32(pb[16:], p.HP)
	return pb[:]
}

func (p *Player) FillFromBytes(reader io.Reader) {
	data := make([]byte, 20)
	_, err := reader.Read(data)
	if err != nil {
		panic(err)
	}
	p.ID = ObjectID(binary.BigEndian.Uint32(data[:4]))
	p.ViewDirection = Direction(binary.BigEndian.Uint32(data[4:8]))
	X := binary.BigEndian.Uint32(data[8:12])
	Y := binary.BigEndian.Uint32(data[12:16])
	p.Position = Vector{float64(X), float64(Y)}
	p.HP = binary.BigEndian.Uint32(data[16:20])
}

type Projectile struct {
	ID            ObjectID
	Rune          rune
	Position      Vector
	Speed         Vector
	CollisionArea CollisionArea
}

func (p Projectile) GetID() ObjectID {
	return p.ID
}

func (p Projectile) GetSpeed() Vector {
	return p.Speed
}

func (p *Projectile) SetSpeed(speed Vector) {
	p.Speed = speed
}

func (p Projectile) GetPosition() Vector {
	return p.Position
}

func (p *Projectile) SetPosition(position Vector) {
	p.Position = position
}

func (p Projectile) GetCollisionArea() CollisionArea {
	return p.CollisionArea
}

func (p Projectile) GetCollisionBox() CollisionBox {
	return p.CollisionArea.ToCollisionBox(p.Position)
}

func (p *Projectile) OnCollision(co CollidableObject) {
	return
}

func (p Projectile) ToString() string {
	return fmt.Sprintf(
		"Projectile %c, Position: %s, Speed: %s",
		p.Rune,
		p.Position.ToString(),
		p.Speed.ToString(),
	)
}
func (p Projectile) ToBytes() []byte {
	pb := [16]byte{}
	binary.BigEndian.PutUint32(pb[:4], uint32(p.ID))
	binary.BigEndian.PutUint32(pb[4:8], uint32(p.Rune))
	binary.BigEndian.PutUint32(pb[8:12], uint32(p.Position.X))
	binary.BigEndian.PutUint32(pb[12:], uint32(p.Position.Y))
	return pb[:]
}

func (p *Projectile) FillFromBytes(reader io.Reader) {
	data := make([]byte, 16, 16)
	_, err := reader.Read(data)
	if err != nil {
		panic(err)
	}
	p.ID = ObjectID(binary.BigEndian.Uint32(data[:4]))
	p.Rune = rune(binary.BigEndian.Uint32(data[4:8]))
	X := binary.BigEndian.Uint32(data[8:12])
	Y := binary.BigEndian.Uint32(data[12:16])
	p.Position = Vector{float64(X), float64(Y)}
}

type PlayerMap map[ObjectID]*Player

func (pm PlayerMap) ToBytes() []byte {
	res := []byte{byte(len(pm))}
	for _, p := range pm {
		res = append(res, p.ToBytes()[:]...)
	}
	return res
}

func (pm PlayerMap) FillFromBytes(reader io.Reader) {
	playerNumberBuff := make([]byte, 1, 1)
	_, err := reader.Read(playerNumberBuff)
	if err != nil {
		panic(err)
	}
	playerNumber := int(playerNumberBuff[0])
	for range playerNumber {
		player := Player{}
		player.FillFromBytes(reader)

		pm[player.ID] = &player
	}
}

type ProjectileMap map[ObjectID]*Projectile

func (pm ProjectileMap) ToBytes() []byte {
	res := []byte{byte(len(pm))}
	for _, p := range pm {
		res = append(res, p.ToBytes()[:]...)
	}
	return res
}

func (pm ProjectileMap) FillFromBytes(reader io.Reader) {
	projectileNuberBuff := make([]byte, 1, 1)
	_, err := reader.Read(projectileNuberBuff)
	if err != nil {
		panic(err)
	}
	projectileNumber := int(projectileNuberBuff[0])
	for range projectileNumber {
		projectile := Projectile{}
		projectile.FillFromBytes(reader)

		pm[projectile.ID] = &projectile
	}
}

type InitializationData struct {
	PlayerID ObjectID
}

func (initData InitializationData) ToBytes() []byte {
	res := [4]byte{}
	binary.BigEndian.PutUint32(res[:4], uint32(initData.PlayerID))
	return res[:]
}

func (initData *InitializationData) FillFromBytes(reader io.Reader) {
	playerIDBuff := [4]byte{}
	_, err := reader.Read(playerIDBuff[:])
	if err != nil {
		panic(err)
	}

	playerID := ObjectID(binary.BigEndian.Uint32(playerIDBuff[:4]))
	initData.PlayerID = playerID
}

func InitializationDataFromBytes(reader io.Reader) InitializationData {
	initializationData := InitializationData{}
	initializationData.FillFromBytes(reader)
	return initializationData
}

type GameState struct {
	Players     PlayerMap
	Projectiles ProjectileMap
	MapObjects  []MapObject
}

func (gs GameState) ToBytes() []byte {
	res := []byte{}

	res = append(res, gs.Players.ToBytes()...)
	res = append(res, gs.Projectiles.ToBytes()...)
	return res
}

func GameStateFromBytes(reader io.Reader) GameState {
	playerMap := PlayerMap{}
	playerMap.FillFromBytes(reader)

	projectileMap := ProjectileMap{}
	projectileMap.FillFromBytes(reader)
	gameState := GameState{Players: playerMap, Projectiles: projectileMap}
	return gameState
}

// TODO: Map objects should be dynamic and passed from server to client on init

type CollisionBox struct {
	BottomLeft Vector
	TopRight   Vector
	IsRigid    bool
}

func (cb *CollisionBox) Add(v Vector) CollisionBox {
	return CollisionBox{BottomLeft: cb.BottomLeft.Add(v), TopRight: cb.TopRight.Add(v), IsRigid: cb.IsRigid}
}

func (cb CollisionBox) IsVectorWithin(v Vector) bool {
	return v.X >= cb.BottomLeft.X &&
		v.X < cb.TopRight.X &&
		v.Y >= cb.BottomLeft.Y &&
		v.Y < cb.TopRight.Y
}

func (cb CollisionBox) IntersectsWithX(other CollisionBox) bool {
	minRightX := min(cb.TopRight.X, other.TopRight.X)
	maxLeftX := max(cb.BottomLeft.X, other.BottomLeft.X)
	return minRightX > maxLeftX
}
func (cb CollisionBox) IntersectsWithY(other CollisionBox) bool {
	minRightY := min(cb.TopRight.Y, other.TopRight.Y)
	maxLeftY := max(cb.BottomLeft.Y, other.BottomLeft.Y)
	return minRightY > maxLeftY
}
func (cb CollisionBox) IntersectsWith(other CollisionBox) bool {
	return cb.IntersectsWithX(other) && cb.IntersectsWithY(other)
}

func (cb CollisionBox) ToString() string {
	return fmt.Sprintf("[%s, %s]", cb.BottomLeft.ToString(), cb.TopRight.ToString())
}

type MapObject struct {
	Position      Vector        `json:"position"`
	CollisionArea CollisionArea `json:"collision_area"`
	IsVisible     bool          `json:"is_visible"`
}

func (mo MapObject) GetPosition() Vector {
	return mo.Position
}

func (mo MapObject) GetCollisionArea() CollisionArea {
	return mo.CollisionArea
}

func (mo MapObject) GetCollisionBox() CollisionBox {
	return mo.CollisionArea.ToCollisionBox(mo.Position)
}

func (mo MapObject) OnCollision(CollidableObject) {
	return
}

func __read_map_objects() []MapObject {
	b, err := os.ReadFile("map.json")
	if err != nil {
		panic(err)
	}
	var mos []MapObject
	err = json.Unmarshal(b, &mos)
	if err != nil {
		panic(err)
	}

	// Map invisible borders
	mos = append(mos,
		MapObject{Position: Vector{X: -1, Y: -1}, CollisionArea: CollisionArea{X: FieldMaxX + 2, Y: 1}},                    // Bottom
		MapObject{Position: Vector{X: -1, Y: FieldMaxY}, CollisionArea: CollisionArea{X: FieldMaxX + 2, Y: FieldMaxY + 2}}, // Top
		MapObject{Position: Vector{X: -1, Y: -1}, CollisionArea: CollisionArea{X: 1, Y: FieldMaxY + 2}},                    // Left
		MapObject{Position: Vector{X: FieldMaxX, Y: -1}, CollisionArea: CollisionArea{X: FieldMaxX + 2, Y: FieldMaxY + 2}}, // Right
	)
	return mos
}

var MapObjects = __read_map_objects()
