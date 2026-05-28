# ChessPuzzleFinder

Simple Go app for finding tactics puzzles in chess games using a UCI chess
engine.

The app runs as a long-lived worker that:

1. Pulls chess games (lists of UCI moves) from an **Azure Storage Queue**.
2. Replays each game through a UCI chess engine (Stockfish) and looks for
   positions where one side missed a clearly winning move (i.e. a tactic).
3. Writes any tactics it finds to a **SQL Server** `Puzzles` table so they
   can be served up later as one-move puzzles.

A position is recorded as a puzzle when the move that was actually played
differs from the engine's best move, the score gap between the engine's
best and second-best lines is at least 300 centipawns, and at least one
of those two lines still has an evaluation within ±600 centipawns (i.e.
the position isn't already obviously decided). The first six plies of
each game are skipped because they are unlikely to contain interesting
tactics. See [`tactics/analyzer.go`](tactics/analyzer.go) for the exact
heuristic.

## Repository layout

```
.
├── cmd/ingestgames/   # main package – the worker entry point
│   ├── main.go        # config + queue loop
│   ├── chessgame.go   # JSON shape of a queue message
│   └── data.go        # SQL Server puzzle writer
├── tactics/           # puzzle-finding logic (UCI driver)
│   ├── analyzer.go    # talks to Stockfish, decides what counts as a puzzle
│   ├── chesspiece.go
│   ├── move.go
│   └── puzzle.go
├── config.json        # sample configuration
├── Dockerfile         # multi-stage container build
└── go.mod / go.sum
```

## Prerequisites

To build and run the app you need:

- **Go** 1.14 or newer (`go version`). The module currently targets Go 1.14
  in `go.mod` but builds with any modern Go release.
- **A UCI chess engine.** The app is developed and tested against
  [Stockfish](https://stockfishchess.org/) and relies on a few
  Stockfish-flavored behaviors beyond plain UCI: the `d` command (which
  responds with a `Fen: …` line) and `setoption name MultiPV value 2`.
  Engines that don't support these may not work without modification.
- **An Azure Storage account** with a Queue that the worker can dequeue
  game messages from.
- **A SQL Server database** with a `Puzzles` table matching the schema
  used in [`cmd/ingestgames/data.go`](cmd/ingestgames/data.go) (columns:
  `CreatedDate`, `LastModifiedDate`, `Position`, `SetupMovedFrom`,
  `SetupMovedTo`, `SetupPiecePromotedTo`, `MovedFrom`, `MovedTo`,
  `PiecePromotedTo`, `IncorrectMovedFrom`, `IncorrectMovedTo`,
  `IncorrectPiecePromotedTo`, `Site`, `GameDate`, `GameUrl`,
  `AssociatedPlayerId`, `BlackPlayerName`, `WhitePlayerName`). The repo
  does not currently ship a `CREATE TABLE` script — you will need to
  define column types yourself (e.g. `Position` as the FEN string,
  `MovedFrom`/`MovedTo` as 2-character squares, `PiecePromotedTo` as a
  nullable integer, `Site` as a short text label, `GameDate` as a
  datetime, etc.).
- **A populated `config.json`.** The worker calls `viper.ReadInConfig()`
  at startup and exits with a fatal error if no `config.json` is found
  next to the binary or in the current working directory. Environment
  variables can override individual values, but they do not remove the
  requirement to have a `config.json` file present.

Docker is optional but supported via the included `Dockerfile`.

## Configuration

Configuration is loaded via [Viper](https://github.com/spf13/viper) from a
`config.json` file. The app looks for `config.json` first next to the
compiled executable and then in the current working directory. Any setting
can also be supplied (or overridden) by an environment variable with the
**same name in upper case**.

| Key                      | Type   | Description                                                                                  |
| ------------------------ | ------ | -------------------------------------------------------------------------------------------- |
| `Environment`            | string | Free-form environment label (e.g. `Development`). Currently read into a flag but not acted on elsewhere; safe to set to anything. |
| `PuzzleDbConnectionString` | string | SQL Server connection string used to insert puzzles.                                       |
| `StorageAccountName`     | string | Azure Storage account that hosts the game queue.                                             |
| `StorageAccountKey`      | string | Shared key for the storage account.                                                          |
| `GameIngestionQueue`     | string | Name of the queue to read game messages from (e.g. `games`).                                 |
| `WorkerCount`            | int    | Number of concurrent workers that dequeue and analyze games.                                 |
| `AnalysisSecondsPerMove` | int    | Wall-clock seconds Stockfish is given per move analyzed.                                     |
| `AnalysisDepth`          | int    | Target UCI search depth used by the analyzer.                                                |
| `EnginePath`             | string | Absolute path to the UCI engine executable (e.g. Stockfish).                                 |

A starter `config.json` is included; fill in the connection string, storage
credentials, and engine path before running:

```json
{
    "Environment": "Development",
    "PuzzleDbConnectionString": "sqlserver://user:pass@host?database=Puzzles",
    "StorageAccountName": "mystorageacct",
    "StorageAccountKey": "<base64-key>",
    "GameIngestionQueue": "games",
    "WorkerCount": 1,
    "AnalysisSecondsPerMove": 12,
    "AnalysisDepth": 20,
    "EnginePath": "C:\\Tools\\Stockfish\\stockfish.exe"
}
```

Equivalent environment variable overrides (handy for containers / CI):

```bash
export ENVIRONMENT=Development
export PUZZLEDBCONNECTIONSTRING="sqlserver://user:pass@host?database=Puzzles"
export STORAGEACCOUNTNAME=mystorageacct
export STORAGEACCOUNTKEY=<base64-key>
export GAMEINGESTIONQUEUE=games
export WORKERCOUNT=1
export ANALYSISSECONDSPERMOVE=12
export ANALYSISDEPTH=20
export ENGINEPATH=/usr/local/bin/stockfish
```

> ⚠️ `config.json` contains secrets when populated. Do not commit a
> filled-in copy; prefer environment variables in shared environments.

### Queue message format

Each Azure Storage Queue message is expected to be a JSON document that
deserializes into the `game` struct in
[`cmd/ingestgames/chessgame.go`](cmd/ingestgames/chessgame.go):

```json
{
    "GameUrl": "https://lichess.org/abcd1234",
    "Site": 0,
    "GameDate": "2020-05-01T12:34:56Z",
    "UCIMoves": ["e2e4", "e7e5", "g1f3", "..."],
    "AssociatedPlayerId": 42,
    "WhitePlayer": "alice",
    "BlackPlayer": "bob"
}
```

`Site` is an integer enum: `0` = `lichess.org`, `1` = `chess.com`.

## Building

### Build locally with `go build`

From the repository root:

```bash
# Download module dependencies (first time only)
go mod download

# Build the ingestgames binary into the current directory
go build ./cmd/ingestgames
```

On Windows this produces `ingestgames.exe`; on Linux/macOS it produces
`ingestgames`. You can also install the binary into `$GOBIN` with:

```bash
go install ./cmd/ingestgames
```

### Build with Docker

The included `Dockerfile` is a multi-stage build that compiles the binary
and installs Stockfish from the Debian package repository into the
runtime image (no manual download required):

```bash
docker build -t chesspuzzlefinder .
```

The resulting image bundles `config.json` and sets `ENGINEPATH` to the
packaged Stockfish binary at `/usr/games/stockfish`, so at runtime you
only need to supply secrets/connection settings.

## Running

### Run locally

1. Make sure a UCI engine (e.g. Stockfish) is installed and that the path
   in `EnginePath` points at the executable.
2. Make sure your `config.json` (or environment variables) has valid
   Azure Storage credentials, a queue name, and a SQL Server connection
   string.
3. Run the worker:

   ```bash
   # From the repository root, using config.json in the cwd:
   go run ./cmd/ingestgames

   # Or run the pre-built binary:
   ./ingestgames          # Linux/macOS
   .\ingestgames.exe      # Windows
   ```

The process will print a banner, log how many workers it started, and then
loop forever dequeueing and analyzing games. Press `Ctrl+C` (or send
`SIGTERM`) to shut down — in-flight workers are canceled cooperatively and
the process exits cleanly.

Typical startup output:

```
-----------------------
- Chess Puzzle Finder -
-----------------------

Processing games from queue "games" with 1 workers
Received message <id> (<n> bytes)
Processing game https://lichess.org/abcd1234
Analyzing a game with 57 moves
Starting /usr/local/bin/stockfish
...
Identified 2 puzzles
Inserted puzzle into database
```

### Run with Docker

Provide the configuration values that aren't baked into the image as
environment variables:

```bash
docker run --rm \
    -e PUZZLEDBCONNECTIONSTRING="sqlserver://user:pass@host?database=Puzzles" \
    -e STORAGEACCOUNTNAME=mystorageacct \
    -e STORAGEACCOUNTKEY=<base64-key> \
    -e GAMEINGESTIONQUEUE=games \
    -e WORKERCOUNT=2 \
    -e ANALYSISSECONDSPERMOVE=12 \
    -e ANALYSISDEPTH=20 \
    chesspuzzlefinder
```

To use a different engine binary or your own `config.json`, mount them
into the container and point `ENGINEPATH` / the working directory at the
mounted path.

### Debugging in VS Code

The repo includes a `.vscode/launch.json` configuration named **Launch**
that runs `cmd/ingestgames` from the workspace root. Open the folder in
VS Code with the Go extension installed, set `config.json` up, then
press **F5**.

## Tuning the analyzer

Two knobs in the configuration control the cost/quality trade-off of
analysis:

- `AnalysisSecondsPerMove` — wall-clock time the engine is allowed to
  think about each candidate position. Higher = better puzzle detection,
  slower throughput.
- `AnalysisDepth` — target search depth at which scores are considered
  trustworthy. Higher = more reliable, slower.

`WorkerCount` controls how many games are processed in parallel; note
that each worker spawns its own Stockfish process, so total CPU usage
scales roughly with `WorkerCount`.

## License

Released under the [MIT License](LICENSE).
