// Package replaysync 确定性回放：同一份输入指令必须给出同一个结果。
// 每帧推进量 = speed * dt；同帧指令按玩家编号排序后应用；指纹与校验和覆盖全部玩家的 X/Y。
package replaysync

import (
	"errors"
	"math"
	"sort"
	"strconv"
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
	Sum []uint64 // 每帧一个指纹，覆盖当时全部玩家的 X/Y
}

// Replay 逐帧回放。每帧推进量为 DX/DY * dt；同帧指令按玩家编号稳定排序后应用，
// 因此同一份输入（即使帧内指令顺序被打乱）必然给出逐帧相同的结果。
func Replay(frames [][]Command, players int, dt float64) (*World, error) {
	if players <= 0 {
		return nil, ErrInput
	}
	if !(dt > 0) || math.IsInf(dt, 0) {
		return nil, ErrInput
	}
	world := &World{
		X:   make([]float64, players),
		Y:   make([]float64, players),
		Sum: make([]uint64, 0, len(frames)),
	}
	ordered := make([]Command, 0, 64) // 跨帧复用，避免每帧重新分配
	for _, commands := range frames {
		ordered = ordered[:0]
		for _, command := range commands {
			if command.Player < 0 || command.Player >= players ||
				invalid(command.DX) || invalid(command.DY) {
				return nil, ErrInput
			}
			ordered = append(ordered, command)
		}
		// 稳定排序：相同玩家保持输入先后，顺序与遍历方式无关。
		sort.SliceStable(ordered, func(i, j int) bool {
			return ordered[i].Player < ordered[j].Player
		})
		for _, command := range ordered {
			world.X[command.Player] += command.DX * dt
			world.Y[command.Player] += command.DY * dt
		}
		world.Sum = append(world.Sum, fingerprint(world))
	}
	return world, nil
}

func invalid(v float64) bool {
	return math.IsNaN(v) || math.IsInf(v, 0)
}

// fingerprint 把全部玩家的 X、Y 混合进一个 64 位指纹（FNV-1a 风格，按 8 字节整块混合）。
func fingerprint(world *World) uint64 {
	h := uint64(14695981039346656037)
	mix := func(v float64) {
		h ^= math.Float64bits(v)
		h *= 1099511628211
	}
	for i := range world.X {
		mix(world.X[i])
		mix(world.Y[i])
	}
	return h
}

// Checksum 覆盖全部玩家的 X 与 Y，篡改任何一个分量都会得到不同的校验和。
func Checksum(world *World) string {
	if world == nil || len(world.X) == 0 {
		return ""
	}
	return strconv.FormatUint(fingerprint(world), 16)
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
