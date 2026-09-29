// Package replaysync 确定性回放：同一份输入指令必须给出同一个结果。
package replaysync

import (
	"cmp"
	"errors"
	"fmt"
	"math"
	"slices"
)

// ErrInput 表示指令不合法。
var ErrInput = errors.New("replaysync: 指令不合法")

// Command 是一条输入指令。
type Command struct {
	Player int
	DX     float64
	DY     float64
}

// World 是回放用的世界状态。
type World struct {
	X   []float64
	Y   []float64
	Sum []uint64
}

// Replay 逐帧回放。每帧推进量为 speed * dt；同帧指令按玩家编号稳定排序后应用；
// 每帧摘要覆盖全部玩家的 X、Y，与指令给出顺序无关。
func Replay(frames [][]Command, players int, dt float64) (*World, error) {
	if players <= 0 {
		return nil, ErrInput
	}
	if dt <= 0 || math.IsNaN(dt) || math.IsInf(dt, 0) {
		return nil, ErrInput
	}
	world := &World{
		X:   make([]float64, players),
		Y:   make([]float64, players),
		Sum: make([]uint64, 0, len(frames)),
	}
	ordered := make([]Command, 0, 64)
	for _, commands := range frames {
		ordered = append(ordered[:0], commands...)
		slices.SortStableFunc(ordered, func(a, b Command) int {
			return cmp.Compare(a.Player, b.Player)
		})
		for _, command := range ordered {
			if command.Player < 0 || command.Player >= players {
				return nil, ErrInput
			}
			world.X[command.Player] += command.DX * dt
			world.Y[command.Player] += command.DY * dt
		}
		world.Sum = append(world.Sum, digest(world))
	}
	return world, nil
}

// digest 按玩家编号顺序把每个玩家的 X、Y 混入哈希，覆盖全部状态。
func digest(world *World) uint64 {
	const prime = 1099511628211
	h := uint64(14695981039346656037)
	for player := range world.X {
		h = (h ^ math.Float64bits(world.X[player])) * prime
		h = (h ^ math.Float64bits(world.Y[player])) * prime
	}
	return h
}

// Checksum 覆盖所有玩家的 X 与 Y，篡改任何一项都会改变结果。
func Checksum(world *World) string {
	if world == nil || len(world.X) == 0 {
		return ""
	}
	return fmt.Sprintf("%016x", digest(world))
}

// Script 生成一段固定脚本：玩家 0..players-1 每帧各走一步，其中第 3 帧有两人撞到同一格。
func Script(frames, players int, speed float64) [][]Command {
	script := make([][]Command, 0, frames)
	for frame := 0; frame < frames; frame++ {
		commands := []Command{}
		for player := 0; player < players; player++ {
			dx := speed
			if frame == 3 && player%2 == 1 {
				dx = -speed
			}
			commands = append(commands, Command{Player: player, DX: dx, DY: 0})
		}
		script = append(script, commands)
	}
	return script
}
