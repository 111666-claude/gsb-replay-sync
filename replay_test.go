package replaysync

import (
	"math"
	"math/rand"
	"reflect"
	"sort"
	"testing"
	"time"
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

// --- 以下为口径修复后的新增用例，既有用例与断言保持不变 ---

// 帧率无关：同样的物理时间，不同帧率下位置必须一致（误差 ≤ 1e-9）。
func TestFrameRateIndependent(t *testing.T) {
	const speed = 60.0
	// 匀速脚本：每帧每个玩家都走 speed，代表同一段物理输入。
	uniform := func(frames int) [][]Command {
		script := make([][]Command, frames)
		for f := range script {
			commands := make([]Command, 4)
			for p := range commands {
				commands[p] = Command{Player: p, DX: speed, DY: speed / 2}
			}
			script[f] = commands
		}
		return script
	}
	cases := []struct{ framesA, framesB, fpsA, fpsB int }{
		{600, 1200, 30, 60},   // 都是 20 秒
		{100, 300, 20, 60},    // 都是 5 秒
		{240, 14400, 12, 720}, // 都是 20 秒
	}
	for _, c := range cases {
		worldA, err := Replay(uniform(c.framesA), 4, 1.0/float64(c.fpsA))
		if err != nil {
			t.Fatalf("Replay 出错：%v", err)
		}
		worldB, err := Replay(uniform(c.framesB), 4, 1.0/float64(c.fpsB))
		if err != nil {
			t.Fatalf("Replay 出错：%v", err)
		}
		for p := range worldA.X {
			if diff := math.Abs(worldA.X[p] - worldB.X[p]); diff > 1e-9 {
				t.Fatalf("fps %d 与 %d 下玩家 %d 的 X 不一致：%v vs %v（差 %v）",
					c.fpsA, c.fpsB, p, worldA.X[p], worldB.X[p], diff)
			}
			if diff := math.Abs(worldA.Y[p] - worldB.Y[p]); diff > 1e-9 {
				t.Fatalf("fps %d 与 %d 下玩家 %d 的 Y 不一致：%v vs %v（差 %v）",
					c.fpsA, c.fpsB, p, worldA.Y[p], worldB.Y[p], diff)
			}
		}
	}
}

// 低帧率不慢：--frames 600 --fps 30（20 秒、60 单位/秒）必须走出 1200。
func TestDistanceMatchesPhysicalTime(t *testing.T) {
	world, err := Replay(Script(600, 1, 60), 1, 1.0/30.0)
	if err != nil {
		t.Fatalf("Replay 出错：%v", err)
	}
	if diff := math.Abs(world.X[0] - 1200); diff > 1e-9 {
		t.Fatalf("30fps 下 20 秒应该走 1200，实际 %v", world.X[0])
	}
}

// 顺序确定：同一帧的指令集合无论如何重排，逐帧指纹与最终状态都必须相同。
func TestCommandOrderDeterministic(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	base := Script(50, 8, 3)
	shuffled := make([][]Command, len(base))
	for i, commands := range base {
		cp := append([]Command(nil), commands...)
		rng.Shuffle(len(cp), func(a, b int) { cp[a], cp[b] = cp[b], cp[a] })
		shuffled[i] = cp
	}
	worldA, err := Replay(base, 8, 1.0/60.0)
	if err != nil {
		t.Fatalf("Replay 出错：%v", err)
	}
	for trial := 0; trial < 20; trial++ {
		worldB, err := Replay(shuffled, 8, 1.0/60.0)
		if err != nil {
			t.Fatalf("Replay 出错：%v", err)
		}
		if !reflect.DeepEqual(worldA.X, worldB.X) || !reflect.DeepEqual(worldA.Y, worldB.Y) {
			t.Fatal("帧内指令重排后状态变了")
		}
		if !reflect.DeepEqual(worldA.Sum, worldB.Sum) {
			t.Fatal("帧内指令重排后逐帧指纹变了")
		}
	}
}

// 不变量：同一输入重复回放，逐帧结果完全相同。
func TestRepeatConsistent(t *testing.T) {
	script := Script(60, 4, 5)
	first, err := Replay(script, 4, 1.0/60.0)
	if err != nil {
		t.Fatalf("Replay 出错：%v", err)
	}
	for i := 0; i < 20; i++ {
		world, err := Replay(script, 4, 1.0/60.0)
		if err != nil {
			t.Fatalf("Replay 出错：%v", err)
		}
		if !reflect.DeepEqual(first.Sum, world.Sum) {
			t.Fatalf("第 %d 次回放指纹不一致", i)
		}
		if Checksum(first) != Checksum(world) {
			t.Fatalf("第 %d 次回放校验和不一致", i)
		}
	}
}

// 校验和覆盖全部状态：篡改任何玩家的 X 或 Y 都必须被发现。
func TestChecksumCoversAllState(t *testing.T) {
	const players = 4
	replay := func() *World {
		world, err := Replay(Script(10, players, 2), players, 1.0/60.0)
		if err != nil {
			t.Fatalf("Replay 出错：%v", err)
		}
		return world
	}
	before := Checksum(replay())
	for p := 0; p < players; p++ {
		world := replay()
		world.X[p] += 0.25
		if Checksum(world) == before {
			t.Fatalf("篡改玩家 %d 的 X 未被发现", p)
		}
		world = replay()
		world.Y[p] += 0.25
		if Checksum(world) == before {
			t.Fatalf("篡改玩家 %d 的 Y 未被发现", p)
		}
	}
}

// 非法输入：越界玩家编号、NaN/Inf 位移、非法 dt 都要报 ErrInput。
func TestInvalidInputRejected(t *testing.T) {
	if _, err := Replay([][]Command{{{Player: -1, DX: 1}}}, 2, 1.0/60.0); err != ErrInput {
		t.Fatalf("负数玩家应该报 ErrInput：%v", err)
	}
	if _, err := Replay([][]Command{{{Player: 0, DX: math.NaN()}}}, 2, 1.0/60.0); err != ErrInput {
		t.Fatalf("NaN 位移应该报 ErrInput：%v", err)
	}
	if _, err := Replay([][]Command{{{Player: 0, DY: math.Inf(1)}}}, 2, 1.0/60.0); err != ErrInput {
		t.Fatalf("Inf 位移应该报 ErrInput：%v", err)
	}
	if _, err := Replay(Script(1, 1, 1), 1, 0); err != ErrInput {
		t.Fatalf("零 dt 应该报 ErrInput：%v", err)
	}
	if _, err := Replay(Script(1, 1, 1), 1, -1.0/60.0); err != ErrInput {
		t.Fatalf("负 dt 应该报 ErrInput：%v", err)
	}
}

// 对照自证：随机 200 份输入脚本，与「按排序后的指令顺序、固定 dt 推进」的
// 参考实现逐帧比对，差异必须为 0。
func TestAgainstReferenceImplementation(t *testing.T) {
	rng := rand.New(rand.NewSource(2026))
	const players = 6
	for trial := 0; trial < 200; trial++ {
		frameCount := 1 + rng.Intn(30)
		script := make([][]Command, frameCount)
		for f := range script {
			n := rng.Intn(players + 1)
			perm := rng.Perm(players)[:n] // 每帧玩家不重复，构成指令集合
			commands := make([]Command, n)
			for i, p := range perm {
				commands[i] = Command{
					Player: p,
					DX:     rng.Float64()*20 - 10,
					DY:     rng.Float64()*20 - 10,
				}
			}
			script[f] = commands
		}
		dt := 1.0 / float64(1+rng.Intn(240))

		// 参考实现：排序后的指令顺序、固定 dt 推进，逐帧记录位置。
		refX := make([]float64, players)
		refY := make([]float64, players)
		for f, commands := range script {
			sorted := append([]Command(nil), commands...)
			sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Player < sorted[j].Player })
			for _, c := range sorted {
				refX[c.Player] += c.DX * dt
				refY[c.Player] += c.DY * dt
			}
			// 用前缀回放拿到第 f 帧结束时的状态，逐帧比对。
			world, err := Replay(script[:f+1], players, dt)
			if err != nil {
				t.Fatalf("Replay 出错：%v", err)
			}
			for p := 0; p < players; p++ {
				if world.X[p] != refX[p] || world.Y[p] != refY[p] {
					t.Fatalf("脚本 %d 第 %d 帧玩家 %d 不一致：(%v,%v) vs 参考 (%v,%v)",
						trial, f, p, world.X[p], world.Y[p], refX[p], refY[p])
				}
			}
		}
	}
}

// 规模：100 万帧、64 个玩家 2 秒内回放完；指纹计算为 O(玩家数)/帧，无每帧分配。
func TestScaleMillionFrames(t *testing.T) {
	const frames, players = 1_000_000, 64
	script := make([][]Command, frames)
	for f := range script {
		script[f] = []Command{{Player: f % players, DX: 1, DY: -1}}
	}
	start := time.Now()
	world, err := Replay(script, players, 1.0/60.0)
	if err != nil {
		t.Fatalf("Replay 出错：%v", err)
	}
	elapsed := time.Since(start)
	if len(world.Sum) != frames {
		t.Fatalf("帧数不对：%d", len(world.Sum))
	}
	if elapsed > 2*time.Second {
		t.Fatalf("100 万帧回放超时：%v", elapsed)
	}
	t.Logf("100 万帧 × 64 玩家回放耗时 %v", elapsed)
}
