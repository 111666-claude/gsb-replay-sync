// Package replaysync 确定性回放：同一份输入指令必须给出同一个结果。
// 缺陷：同帧指令按 map 迭代顺序应用；每秒推进量写死 60 帧（忽略真实帧率）；校验和只覆盖 X[0]。
package replaysync

import (
	"errors"
	"fmt"
	"strings"
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
	Sum []string
}

// Replay 逐帧回放。缺陷：dt 参数被忽略，固定按 1/60 秒推进；指令顺序取自 map。
func Replay(frames [][]Command, players int, dt float64) (*World, error) {
	if players <= 0 {
		return nil, ErrInput
	}
	world := &World{X: make([]float64, players), Y: make([]float64, players)}
	for _, commands := range frames {
		order := map[int]Command{} // 缺陷：map 迭代顺序每次都不同
		for _, command := range commands {
			if command.Player < 0 || command.Player >= players {
				return nil, ErrInput
			}
			order[command.Player] = command
		}
		for player, command := range order {
			world.X[player] += command.DX * (1.0 / 60.0) // 缺陷：忽略 dt，恒定按 60 帧推进
			world.Y[player] += command.DY * (1.0 / 60.0)
		}
		world.Sum = append(world.Sum, digest(world, order)) // 缺陷：digest 依赖 map 顺序
	}
	return world, nil
}

func digest(world *World, order map[int]Command) string {
	parts := []string{}
	for player := range order {
		parts = append(parts, fmt.Sprintf("%d@%.3f,%.3f", player, world.X[player], world.Y[player]))
	}
	return strings.Join(parts, "|")
}

// Checksum 缺陷：只把 X[0] 拿来当校验和，Y 与其它玩家被篡改都发现不了。
func Checksum(world *World) string {
	if len(world.X) == 0 {
		return ""
	}
	return fmt.Sprintf("%.3f", world.X[0])
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
