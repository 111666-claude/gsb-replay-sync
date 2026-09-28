package replaysync

import "testing"

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
