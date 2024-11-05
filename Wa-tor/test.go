package main

import (
	//"fmt"
	"image/color"
	"log"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// Constants for grid and window dimensions
const (
	xdim        = 100                // Number of cells in the x direction
	ydim        = 100                // Number of cells in the y direction
	windowXSize = 800                // Width of the window in pixels
	windowYSize = 600                // Height of the window in pixels
	cellXSize   = windowXSize / xdim // Width of each cell in pixels
	cellYSize   = windowYSize / ydim // Height of each cell in pixels
)

// Game struct representing the state of the game
// It contains a grid where each cell can have a color representing its state
// (e.g., empty, fish, or shark).
type Game struct {
	grid [xdim][ydim]color.Color
}

// Shark struct representing a shark entity
// Contains information such as position, energy level, and breed timer
type Shark struct {
	x, y     int // Position of the shark on the grid
	starve   int // Starve level of the shark
	breedTimer int // Timer for when the shark can reproduce
}

// Fish struct representing a fish entity
// Contains information such as position and breed timer
type Fish struct {
	x, y       int // Position of the fish on the grid
	breedTimer int // Timer for when the fish can reproduce
}

// Update function, called every frame to update the game state
// Currently, no dynamic updates are happening in this simple version
func (g *Game) Update() error {
	


	return nil 
}

// Draw function, called every frame to render the game screen
// It fills the screen with black and then draws each cell with its assigned color
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black) // Clear the screen with black color

	// Iterate over each cell in the grid
	for i := 0; i < xdim; i++ {
		for k := 0; k < ydim; k++ {
			// Calculate the position of the current cell in pixels
			x := i * cellXSize
			y := k * cellYSize
			rectColor := g.grid[i][k] // Get the color of the current cell

			// Draw the cell as a rectangle with the specified color
			ebitenutil.DrawRect(screen, float64(x), float64(y), float64(cellXSize), float64(cellYSize), rectColor)
		}
	}
}

// Layout function, called to set the screen size
// It returns the dimensions of the window, which remains constant
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return windowXSize, windowYSize
}

// NewGame function initializes a new game instance with a grid of cells
// The grid is initialized with alternating colors (green and blue) to represent fish and empty spaces
func NewGame() *Game {
	game := &Game{}

	// Initialize grid
	for i := 0; i < xdim; i++ {
		for k := 0; k < ydim; k++ {
			randomNum := rand.Intn(30) + 1 // Random number between 1 and 30

			if randomNum >= 20 && randomNum <= 25 {
				game.grid[i][k] = color.RGBA{0, 255, 0, 255} // Green for fish
			} else if randomNum > 25 && randomNum <= 30 {
				game.grid[i][k] = color.RGBA{255, 0, 0, 255} // Red for shark
			} else {
				game.grid[i][k] = color.RGBA{0, 0, 255, 255} // Blue for empty
			}
		}
	}

	return game
}

// Main function, entry point of the program
// It creates a new game, sets up the window, and starts the game loop
func main() {
	game := NewGame() // Create a new game instance

	// Set the window size and title
	ebiten.SetWindowSize(windowXSize, windowYSize)
	ebiten.SetWindowTitle("Ebiten Wa-Tor World")

	// Run the game loop, which will call Update and Draw repeatedly
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err) // Log any errors that occur
	}
}
