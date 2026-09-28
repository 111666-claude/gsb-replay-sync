# replay-sync

确定性回放：同一份输入指令与同一个帧率，必须给出同一个结果。只用 Go 标准库。

```
go test ./...
go run ./cmd/replay-sync --frames 600 --fps 30
go run ./cmd/replay-sync --frames 60 --repeat 20
go run ./cmd/replay-sync --frames 10 --tamper y
```

## 口径（README 为准）

- **速度与帧率无关**：每帧推进量必须是 `speed * dt`（dt = 1/fps）。
  `--frames 600 --fps 30`（20 秒）按口径是 1200（60 单位/秒），现在忽略 dt 固定按 1/60 算，给 600
  —— 玩家看到的就是「低帧率下跑得慢一半」。
- **同帧指令顺序必须确定**：同一帧内的指令要按玩家编号（或指令序号）排序后应用，
  不许用 map 迭代顺序。`--frames 60 --repeat 20` 按口径是 `distinct=1 consistent=true`，
  现在每次运行顺序都不同，给 `distinct>1 consistent=false`。
- **校验和要覆盖全部状态**：`--frames 10 --tamper y` 按口径是 `detected=true`，
  现在只算 X[0]，篡改 Y 也给 `detected=false`。
- 不变量：同一输入重复回放逐帧结果相同；不同帧率下「同样的物理时间」给出同样的位置（误差 ≤ 1e-9）；
  越界玩家编号与非法帧要报 ErrInput。
- 规模：100 万帧、64 个玩家 2 秒内回放完，内存 O(玩家数)；不许每次回放都重建整份脚本。

## 现在的行为

- `world.X[player] += command.DX * (1.0/60.0)`：dt 参数收下就丢，30fps 下移动速度只有一半。
- 同帧指令放进 `map[int]Command` 再 range：顺序随机，digest 依赖这个顺序。
- `Checksum` 只看 `world.X[0]`，Y 与其它玩家被改都不报。

## 计划新增的能力

```
--frames N / --players N / --fps N
--repeat N        重复回放并比较结果
--tamper x|y      篡改状态后检查校验和
```

## 输出

```
frames=600 fps=30 distance=600.000
repeat=20 distinct=1 consistent=true
tamper=y detected=true
```

## 目录

```
replay.go              回放与校验和
cmd/replay-sync        命令行入口
replay_test.go         go test 用例
```
