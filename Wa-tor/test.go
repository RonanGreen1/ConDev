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
	grid [xdim][ydim]Entity
	fish []Fish
	shark []Shark
}

// Entity interface for all game entities (Fish)
type Entity interface {
	GetType() string
	GetPosition() (int, int)
	SetPosition(x, y int)
}

// Shark struct representing a shark entity
// Contains information such as position, energy level, and breed timer
type Shark struct {
	x, y     int // Position of the shark on the grid
	//starve   int // Starve level of the shark
	breedTimer int // Timer for when the shark can reproduce
}

func (s *Shark) GetType() string {
	return "shark"
}

func (s *Shark) GetPosition() (int, int) {
	return s.x, s.y
}

func (s *Shark) SetPosition(x, y int) {
	s.x = x
	s.y = y
}

// Fish struct representing a fish entity
// Contains information such as position and breed timer
type Fish struct {
	x, y       int // Position of the fish on the grid
	breedTimer int // Timer for when the fish can reproduce
}

func (f *Fish) GetType() string {
	return "fish"
}

func (f *Fish) GetPosition() (int, int) {
	return f.x, f.y
}

func (f *Fish) SetPosition(x, y int) {
	f.x = x
	f.y = y
}

// Update function, called every frame to update the game state
// Currently, no dynamic updates are happening in this simple version
func (g *Game) Update() error {
	moved := false
	for i := range g.shark {
		shark := &g.shark[i]
		x, y := shark.GetPosition()
		
		for i := 4; i > 0; i-- {
			// Random movement direction: 0 = north, 1 = south, 2 = east, 3 = west
			direction := i
			newX, newY := x, y
			switch direction {
			case 0: // North
				if y > 0 {
					newY = y - 1
				}
			case 1: // South
				if y < ydim-1 {
					newY = y + 1
				}
			case 2: // East
				if x < xdim-1 {
					newX = x + 1
				}
			case 3: // West
				if x > 0 {
					newX = x - 1
				}
			}

			// Check if the new position is empty
			if g.grid[newX][newY].GetType() == "fish" {
				// Move the shark to the new position
				g.grid[x][y] = nil          // Clear old position
				shark.SetPosition(newX, newY)
				g.grid[newX][newY] = shark // Set new position
				shark.breedTimer++ 		// Increment breed timer
				for i := range g.fish {
					if g.fish[i].x == newX && g.fish[i].y == newY {
						g.fish = append(g.fish[:i], g.fish[i+1:]...)
						break
					}
				}
				if shark.breedTimer == 10 {
					shark.breedTimer = 0
					shark := Shark{x: x, y: y, breedTimer: 0}
					g.grid[x][y] = &shark
					g.shark = append(g.shark, shark)
				}
				moved = true
				break
			}
		}
	

		if !moved {
			for i := 4; i > 0; i-- {
				// Random movement direction: 0 = north, 1 = south, 2 = east, 3 = west
				direction := rand.Intn(4)
				newX, newY := x, y
				switch direction {
				case 0: // North
					if y > 0 {
						newY = y - 1
					}
				case 1: // South
					if y < ydim-1 {
						newY = y + 1
					}
				case 2: // East
					if x < xdim-1 {
						newX = x + 1
					}
				case 3: // West
					if x > 0 {
						newX = x - 1
					}
				}
	
				// Check if the new position is empty
				if g.grid[newX][newY] == nil {
					// Move the shark to the new position
					g.grid[x][y] = nil          // Clear old position
					shark.SetPosition(newX, newY)
					g.grid[newX][newY] = shark // Set new position
					shark.breedTimer++ 		// Increment breed timer
					if shark.breedTimer == 10 {
						shark.breedTimer = 0
						shark := Shark{x: x, y: y, breedTimer: 0}
						g.grid[x][y] = &shark
						g.shark = append(g.shark, shark)
					}
					moved = true
					break
				}
			}
		}
	}


		for i := range g.fish {
			fish := &g.fish[i]
			x, y := fish.GetPosition()
			
			for i := 4; i > 0; i-- {
				// Random movement direction: 0 = north, 1 = south, 2 = east, 3 = west
				direction := rand.Intn(4)

				newX, newY := x, y
				switch direction {
				case 0: // North
					if y > 0 {
						newY = y - 1
					}
				case 1: // South
					if y < ydim-1 {
						newY = y + 1
					}
				case 2: // East
					if x < xdim-1 {
						newX = x + 1
					}
				case 3: // West
					if x > 0 {
						newX = x - 1
					}
				}
	
				// Check if the new position is empty
				if g.grid[newX][newY] == nil {
					// Move the fish to the new position
					g.grid[x][y] = nil          // Clear old position
					fish.SetPosition(newX, newY)
					g.grid[newX][newY] = fish // Set new position
					fish.breedTimer++ 		// Increment breed timer
					if fish.breedTimer == 5 {
						fish.breedTimer = 0
						fish := Fish{x: x, y: y, breedTimer: 0}
						g.grid[x][y] = &fish
						g.fish = append(g.fish, fish)
					}
					break
			}
		}

		}
			
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

			// Determine the color based on the entity in the cell
			var rectColor color.Color
			if entity := g.grid[i][k]; entity != nil {
				switch entity.GetType() {
				case "fish":
					rectColor = color.RGBA{0, 255, 0, 255} // Green for fish
				case "shark":
					rectColor = color.RGBA{255, 0, 0, 255} // Red for shark
				}
			} else {
				rectColor = color.RGBA{0, 0, 255, 255} // Blue for empty
			}

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

	// Initialize grid with random fish or empty spaces
	for i := 0; i < xdim; i++ {
		for k := 0; k < ydim; k++ {
			randomNum := rand.Intn(30) + 1 // Random number between 1 and 30

			if randomNum >= 20 && randomNum <= 25 {
				// Create and place a fish
				fish := Fish{x: i, y: k, breedTimer: 0}
				game.grid[i][k] = &fish
				game.fish = append(game.fish, fish)
			} else if randomNum >= 26 && randomNum <= 30 {
				// Create and place a fish
				shark := Shark{x: i, y: k, breedTimer: 0}
				game.grid[i][k] = &shark
				game.shark = append(game.shark, shark)
			} else {
				// Leave the cell empty
				game.grid[i][k] = nil
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