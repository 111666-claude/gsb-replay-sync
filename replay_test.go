package replaysync

import (
	"math/rand"
	"sort"
	"testing"
)

func TestReplayRuns(t *testing.T) {
	world, err := Replay(Script(10, 2, 1), 2, 1.0/60.0)
	if err != nil {
		t.Fatalf("Replay 出错：%v", err)
	}
	if len(world.Sum) != 10 {
		t.Fatalf("帧数不对：%d", len(world.Sum))
	}
}

func TestPlayersMoveForward(t *testing.T) {
	world, _ := Replay(Script(6, 1, 3), 1, 1.0/60.0)
	if world.X[0] <= 0 {
		t.Fatalf("玩家应该向前移动：%v", world.X)
	}
}

func TestBadPlayerRejected(t *testing.T) {
	if _, err := Replay([][]Command{{{Player: 5, DX: 1}}}, 2, 1.0/60.0); err != ErrInput {
		t.Fatalf("越界玩家应该报错：%v", err)
	}
	if _, err := Replay(nil, 0, 0); err != ErrInput {
		t.Fatalf("零玩家应该报错：%v", err)
	}
}

func TestChecksumNonEmpty(t *testing.T) {
	world, _ := Replay(Script(3, 1, 1), 1, 1.0/60.0)
	if Checksum(world) == "" {
		t.Fatal("校验和不该为空")
	}
}

func TestFrameRateIndependent(t *testing.T) {
	// 同一段物理输入：每个玩家每帧匀速 60 单位/秒。
	uniform := func(frames, players int) [][]Command {
		script := make([][]Command, frames)
		for frame := range script {
			commands := make([]Command, players)
			for player := range commands {
				commands[player] = Command{Player: player, DX: 60, DY: -30}
			}
			script[frame] = commands
		}
		return script
	}
	slow, err := Replay(uniform(600, 4), 4, 1.0/30.0) // 20 秒 @30fps
	if err != nil {
		t.Fatalf("Replay 出错：%v", err)
	}
	fast, err := Replay(uniform(1200, 4), 4, 1.0/60.0) // 20 秒 @60fps
	if err != nil {
		t.Fatalf("Replay 出错：%v", err)
	}
	for player := range slow.X {
		if d := slow.X[player] - fast.X[player]; d < -1e-9 || d > 1e-9 {
			t.Fatalf("玩家 %d 的 X 与帧率相关：%v vs %v", player, slow.X[player], fast.X[player])
		}
		if d := slow.Y[player] - fast.Y[player]; d < -1e-9 || d > 1e-9 {
			t.Fatalf("玩家 %d 的 Y 与帧率相关：%v vs %v", player, slow.Y[player], fast.Y[player])
		}
	}
}

func TestCommandOrderDeterministic(t *testing.T) {
	forward := [][]Command{{
		{Player: 2, DX: 1, DY: 2},
		{Player: 0, DX: 3, DY: 4},
		{Player: 1, DX: 5, DY: 6},
		{Player: 0, DX: 7, DY: 8},
	}}
	shuffled := [][]Command{{
		{Player: 0, DX: 3, DY: 4},
		{Player: 0, DX: 7, DY: 8},
		{Player: 1, DX: 5, DY: 6},
		{Player: 2, DX: 1, DY: 2},
	}}
	a, err := Replay(forward, 3, 1.0/60.0)
	if err != nil {
		t.Fatalf("Replay 出错：%v", err)
	}
	b, err := Replay(shuffled, 3, 1.0/60.0)
	if err != nil {
		t.Fatalf("Replay 出错：%v", err)
	}
	if Checksum(a) != Checksum(b) {
		t.Fatalf("同帧指令重排后校验和变了：%s vs %s", Checksum(a), Checksum(b))
	}
	for player := range a.X {
		if a.X[player] != b.X[player] || a.Y[player] != b.Y[player] {
			t.Fatalf("同帧指令重排后位置变了：玩家 %d", player)
		}
	}
	for run := 0; run < 20; run++ {
		again, err := Replay(forward, 3, 1.0/60.0)
		if err != nil {
			t.Fatalf("Replay 出错：%v", err)
		}
		if Checksum(again) != Checksum(a) {
			t.Fatalf("重复回放结果不一致：%s vs %s", Checksum(again), Checksum(a))
		}
	}
}

func TestChecksumCoversAllPlayers(t *testing.T) {
	const players = 4
	world, err := Replay(Script(10, players, 1), players, 1.0/60.0)
	if err != nil {
		t.Fatalf("Replay 出错：%v", err)
	}
	base := Checksum(world)
	for player := 0; player < players; player++ {
		saved := world.X[player]
		world.X[player] += 1
		if Checksum(world) == base {
			t.Fatalf("篡改玩家 %d 的 X 未被发现", player)
		}
		world.X[player] = saved

		saved = world.Y[player]
		world.Y[player] += 1
		if Checksum(world) == base {
			t.Fatalf("篡改玩家 %d 的 Y 未被发现", player)
		}
		world.Y[player] = saved
	}
}

func TestInvalidDTRejected(t *testing.T) {
	if _, err := Replay(Script(1, 1, 1), 1, 0); err != ErrInput {
		t.Fatalf("非法 dt 应该报错：%v", err)
	}
	if _, err := Replay(Script(1, 1, 1), 1, -1.0/60.0); err != ErrInput {
		t.Fatalf("负 dt 应该报错：%v", err)
	}
}

// 对照自证：随机脚本与「按玩家编号稳定排序、固定 dt 推进」的参考实现逐帧比对。
func TestAgainstReference(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for trial := 0; trial < 200; trial++ {
		players := 1 + rng.Intn(8)
		frames := 1 + rng.Intn(50)
		dt := 1.0 / float64(1+rng.Intn(120))
		script := make([][]Command, frames)
		for frame := 0; frame < frames; frame++ {
			count := rng.Intn(2 * players)
			commands := make([]Command, 0, count)
			for i := 0; i < count; i++ {
				commands = append(commands, Command{
					Player: rng.Intn(players),
					DX:     rng.Float64()*20 - 10,
					DY:     rng.Float64()*20 - 10,
				})
			}
			script[frame] = commands
		}

		// 参考实现：稳定排序后按固定 dt 推进，逐帧记录位置。
		refX := make([]float64, players)
		refY := make([]float64, players)
		type framePos struct {
			x []float64
			y []float64
		}
		history := make([]framePos, 0, frames)
		for _, commands := range script {
			ordered := append([]Command(nil), commands...)
			sort.SliceStable(ordered, func(i, j int) bool {
				return ordered[i].Player < ordered[j].Player
			})
			for _, command := range ordered {
				refX[command.Player] += command.DX * dt
				refY[command.Player] += command.DY * dt
			}
			history = append(history, framePos{
				x: append([]float64(nil), refX...),
				y: append([]float64(nil), refY...),
			})
		}

		// 逐帧比对：前缀回放的结果必须与参考实现该帧完全一致。
		for frame := 0; frame < frames; frame++ {
			world, err := Replay(script[:frame+1], players, dt)
			if err != nil {
				t.Fatalf("第 %d 份脚本第 %d 帧 Replay 出错：%v", trial, frame, err)
			}
			for player := 0; player < players; player++ {
				if world.X[player] != history[frame].x[player] || world.Y[player] != history[frame].y[player] {
					t.Fatalf("第 %d 份脚本第 %d 帧玩家 %d 与参考实现不一致：(%v,%v) vs (%v,%v)",
						trial, frame, player,
						world.X[player], world.Y[player],
						history[frame].x[player], history[frame].y[player])
				}
			}
		}

		// 同一输入重复回放逐帧摘要完全相同。
		first, err := Replay(script, players, dt)
		if err != nil {
			t.Fatalf("Replay 出错：%v", err)
		}
		second, err := Replay(script, players, dt)
		if err != nil {
			t.Fatalf("Replay 出错：%v", err)
		}
		if len(first.Sum) != len(second.Sum) {
			t.Fatalf("第 %d 份脚本帧数不一致", trial)
		}
		for frame := range first.Sum {
			if first.Sum[frame] != second.Sum[frame] {
				t.Fatalf("第 %d 份脚本第 %d 帧摘要前后不一致", trial, frame)
			}
		}
	}
}

// 规模：100 万帧、64 个玩家。
func BenchmarkReplayMillionFrames(b *testing.B) {
	const frames = 1000000
	const players = 64
	script := make([][]Command, frames)
	for frame := 0; frame < frames; frame++ {
		script[frame] = []Command{{Player: frame % players, DX: 1, DY: -1}}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Replay(script, players, 1.0/60.0); err != nil {
			b.Fatalf("Replay 出错：%v", err)
		}
	}
}
