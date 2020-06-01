package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/mjrousos/ChessPuzzleFinder/tactics"
	"github.com/spf13/viper"

	// Preface this with an _ since it is needed at runtime for
	// loading the mssql driver but appears (to the compiler) to
	// not be used.
	_ "github.com/denisenkom/go-mssqldb"
)

const insertPuzzleQuery = `INSERT INTO Puzzles (
	CreatedDate, 
	LastModifiedDate, 
	Position, 
	SetupMovedFrom, 
	SetupMovedTo, 
	SetupPiecePromotedTo, 
	MovedFrom, 
	MovedTo, 
	PiecePromotedTo, 
	IncorrectMovedFrom, 
	IncorrectMovedTo, 
	IncorrectPiecePromotedTo, 
	Site, 
	GameDate, 
	GameUrl, 
	AssociatedPlayerId, 
	BlackPlayerName, 
	WhitePlayerName)
VALUES (
	@currentDate,
	@currentDate,
	@position,
	@setupMovedFrom,
	@setupMovedTo,
	@setupPiecePromotedTo,
	@movedFrom,
	@movedTo,
	@piecePromotedTo,
	@incorrectMovedFrom,
	@incorrectMovedTo,
	@incorrectPiecePromotedTo,
	@site,
	@gameDate,
	@gameUrl,
	@associatedPlayerId,
	@blackPlayerName,
	@whitePlayerName
);`

func writePuzzlesToDatabase(ctx context.Context, game game, puzzles []tactics.Puzzle) {
	if puzzles == nil || len(puzzles) == 0 {
		return
	}

	db, err := sql.Open("sqlserver", viper.GetString("PuzzleDbConnectionString"))
	if err != nil {
		log.Fatalln("Error connecting to database: ", err)
	}

	defer db.Close()

	for _, puzzle := range puzzles {
		select {
		case <-ctx.Done():
			// Context canceled
			return
		default:
			var promotedToArg, setupPromotedToArg, incorrectPromotedToArg sql.NamedArg

			// This would *really* benefit from a ternary operator
			if puzzle.CorrectMove.PiecePromotedTo == tactics.WhiteKing {
				promotedToArg = sql.Named("piecePromotedTo", nil)
			} else {
				promotedToArg = sql.Named("piecePromotedTo", int(puzzle.CorrectMove.PiecePromotedTo))
			}

			if puzzle.SetupMove.PiecePromotedTo == tactics.WhiteKing {
				setupPromotedToArg = sql.Named("setupPiecePromotedTo", nil)
			} else {
				setupPromotedToArg = sql.Named("setupPiecePromotedTo", int(puzzle.SetupMove.PiecePromotedTo))
			}

			if puzzle.IncorrectMove.PiecePromotedTo == tactics.WhiteKing {
				incorrectPromotedToArg = sql.Named("incorrectPiecePromotedTo", nil)
			} else {
				incorrectPromotedToArg = sql.Named("incorrectPiecePromotedTo", int(puzzle.IncorrectMove.PiecePromotedTo))
			}

			siteArg := sql.Named("site", nil)
			if game.Site == 0 {
				siteArg = sql.Named("site", "lichess.org")
			} else if game.Site == 1 {
				siteArg = sql.Named("site", "chess.com")
			}

			_, err := db.ExecContext(
				ctx,
				insertPuzzleQuery,
				sql.Named("currentDate", time.Now()),
				sql.Named("position", puzzle.Position),
				sql.Named("setupMovedFrom", puzzle.SetupMove.MovedFrom),
				sql.Named("setupMovedTo", puzzle.SetupMove.MovedTo),
				setupPromotedToArg,
				sql.Named("movedFrom", puzzle.CorrectMove.MovedFrom),
				sql.Named("movedTo", puzzle.CorrectMove.MovedTo),
				promotedToArg,
				sql.Named("incorrectMovedFrom", puzzle.IncorrectMove.MovedFrom),
				sql.Named("incorrectMovedTo", puzzle.IncorrectMove.MovedTo),
				incorrectPromotedToArg,
				siteArg,
				sql.Named("gameDate", game.Gamedate),
				sql.Named("gameUrl", game.GameURL),
				sql.Named("associatedPlayerId", game.Associatedplayerid),
				sql.Named("blackPlayerName", game.Blackplayername),
				sql.Named("whitePlayerName", game.Whiteplayername),
			)
			if err != nil {
				log.Fatalln("Error inserting puzzle into database: ", err)
			}
			log.Println("Inserted puzzle into database")
		}
	}
}
