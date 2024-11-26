package main

import (
	//"fmt"

	"image/color"
	"log"
	"math/rand"
	"sort"
	"time"
	"encoding/csv"
    "os"
    "strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// Constants for grid and window dimensions
const (
	xdim        = 50                // Number of cells in the x direction
	ydim        = 50                // Number of cells in the y direction
	windowXSize = 800                // Width of the window in pixels
	windowYSize = 600                // Height of the window in pixels
	cellXSize   = windowXSize / xdim // Width of each cell in pixels
	cellYSize   = windowYSize / ydim // Height of each cell in pixels
)


// Game struct representing the state of the game
// It contains a grid where each cell can have a color representing its state
// (e.g., empty, fish, or shark).
type Game struct {
	grid  [xdim][ydim]Entity
	fish  []Fish
	shark []Shark
	startTime time.Time
	simComplete bool // Track if the simulation is complete
	totalFrames  int
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
	x, y       int // Position of the shark on the grid
	starve     int // Starve level of the shark
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

func (g *Game) StartSimulation() {
    g.startTime = time.Now()
    g.totalFrames = 0
}

func (g *Game) RecordFrame() {
    g.totalFrames++
}

// Call this after the simulation ends to calculate the FPS
func (g *Game) CalculateAverageFPS() float64 {
    elapsedTime := time.Since(g.startTime).Seconds()
    if elapsedTime > 0 {
        return float64(g.totalFrames) / elapsedTime
    }
    return 0.0
}

// Update function, called every frame to update the game state
// Currently, no dynamic updates are happening in this simple version
func (g *Game) Update() error {

	g.RecordFrame()

	if time.Since(g.startTime) > 10*time.Second {
        g.simComplete = true
        avgFPS := g.CalculateAverageFPS()
        writeSimulationDataToCSV("simulation_results.csv", g, 1, avgFPS)
        return nil
    }
	
	for i := range g.fish {
		fish := &g.fish[i]
		x, y := fish.GetPosition()

		for i := 4; i > 0; i-- {
			// Random movement direction: 0 = north, 1 = south, 2 = east, 3 = west
			direction := rand.Intn(i)

			newX, newY := x, y
			switch direction {
			case 0: // North
				if y > 0 {
					newY = y - 1
				} else {
					newY = ydim - 1 // Wrap to bottom
				}
			case 1: // South
				if y < ydim-1 {
					newY = y + 1
				} else {
					newY = 0 // Wrap to top
				}
			case 2: // East
				if x < xdim-1 {
					newX = x + 1
				} else {
					newX = 0 // Wrap to left
				}
			case 3: // West
				if x > 0 {
					newX = x - 1
				} else {
					newX = xdim - 1 // Wrap to right
				}
			}

			// Ensure the new position is within bounds
			if newX >= 0 && newX < xdim && newY >= 0 && newY < ydim {
				// Check if the new position is empty
				if g.grid[newX][newY] == nil {
					// Move the fish to the new position
					g.grid[x][y] = nil // Clear old position
					fish.SetPosition(newX, newY)
					g.grid[newX][newY] = fish // Set new position
					fish.breedTimer++         // Increment breed timer
					if fish.breedTimer == 5 {
						fish.breedTimer = 0
						newFish := &Fish{x: x, y: y, breedTimer: 0}
						g.grid[x][y] = newFish
						g.fish = append(g.fish, *newFish)
					}
					break
				}
			}
		}
	}

	removedShark := []int{}
	newSharks := []Shark{}
	removedFish := []int{}
	sharkCount := len(g.shark) // Record initial number of sharks to avoid infinite loop during iteration

	for i := 0; i < sharkCount; i++ {
		moved := false
		shark := &g.shark[i]
		x, y := shark.GetPosition()

		// Try to move to a position occupied by a fish first
		for j := 0; j < 4; j++ {
			// Random movement direction: 0 = north, 1 = south, 2 = east, 3 = west
			direction := rand.Intn(4)

			newX, newY := x, y
			switch direction {
				case 0: // North
				if y > 0 {
					newY = y - 1
				} else {
					newY = ydim - 1 // Wrap to bottom
				}
			case 1: // South
				if y < ydim-1 {
					newY = y + 1
				} else {
					newY = 0 // Wrap to top
				}
			case 2: // East
				if x < xdim-1 {
					newX = x + 1
				} else {
					newX = 0 // Wrap to left
				}
			case 3: // West
				if x > 0 {
					newX = x - 1
				} else {
					newX = xdim - 1 // Wrap to right
				}
			}

			// Ensure the new position is within bounds
			if newX >= 0 && newX < xdim && newY >= 0 && newY < ydim {
				// Check if the new position is occupied by a fish
				if g.grid[newX][newY] != nil && g.grid[newX][newY].GetType() == "fish" {
					// Move the shark to the new position
					g.grid[x][y] = nil // Clear old position
					shark.SetPosition(newX, newY)
					g.grid[newX][newY] = shark // Set new position
					shark.starve = 0
					shark.breedTimer++         // Increment breed timer
					if shark.breedTimer == 5 {
						shark.breedTimer = 0
						newShark := Shark{x: x, y: y, breedTimer: 0, starve: 0}
						g.grid[x][y] = &newShark
						newSharks = append(newSharks, newShark)
					}
					// Mark the fish for removal
					for j, fish := range g.fish {
						if fish.x == newX && fish.y == newY {
							removedFish = append(removedFish, j)
							break
						}
					}
					moved = true
					break
				}
			}
		}

		// If shark didn't move to eat a fish, try to move to an empty cell
		if !moved {
			for j := 0; j < 4; j++ {
				// Random movement direction: 0 = north, 1 = south, 2 = east, 3 = west
				direction := rand.Intn(4)

				newX, newY := x, y
				switch direction {
					case 0: // North
					if y > 0 {
						newY = y - 1
					} else {
						newY = ydim - 1 // Wrap to bottom
					}
				case 1: // South
					if y < ydim-1 {
						newY = y + 1
					} else {
						newY = 0 // Wrap to top
					}
				case 2: // East
					if x < xdim-1 {
						newX = x + 1
					} else {
						newX = 0 // Wrap to left
					}
				case 3: // West
					if x > 0 {
						newX = x - 1
					} else {
						newX = xdim - 1 // Wrap to right
					}
				}

				// Ensure the new position is within bounds
				if newX >= 0 && newX < xdim && newY >= 0 && newY < ydim {
					// Check if the new position is empty
					if g.grid[newX][newY] == nil {
						// Move the shark to the new position
						g.grid[x][y] = nil // Clear old position
						shark.SetPosition(newX, newY)
						g.grid[newX][newY] = shark // Set new position
						shark.starve++
						if shark.starve == 5 {
							g.grid[newX][newY] = nil
							removedShark = append(removedShark, i)
						}
						shark.breedTimer++         // Increment breed timer
						if shark.breedTimer == 6 {
							shark.breedTimer = 0
							newShark := Shark{x: x, y: y, breedTimer: 0, starve: 0}
							g.grid[x][y] = &newShark
							newSharks = append(newSharks, newShark)
						}
						moved = true
						break
					}
				}
			}
		}
	}

	// Remove fish that were eaten
	sort.Sort(sort.Reverse(sort.IntSlice(removedFish))) // Sort in reverse order to remove elements from the end first, avoiding index shift issues
	for _, index := range removedFish { // Iterate over the sorted indices and remove the fish
		if index < len(g.fish) {
			g.fish = append(g.fish[:index], g.fish[index+1:]...)
		}
	}

	// Remove sharks that starved
	sort.Sort(sort.Reverse(sort.IntSlice(removedShark))) // Sort in reverse order to remove elements from the end first, avoiding index shift issues
	for _, index := range removedShark { // Iterate over the sorted indices and remove the sharks
		if index < len(g.shark) {
			g.shark = append(g.shark[:index], g.shark[index+1:]...)
		}
	}

	// Add new sharks after iteration
	g.shark = append(g.shark, newSharks...)

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
					rectColor = color.RGBA{0, 221, 255, 1} // Green for fish
				case "shark":
					rectColor = color.RGBA{190, 44,190, 1} // Red for shark
				}
			} else {
				rectColor = color.RGBA{0, 0, 0, 0} // Blue for empty
			}

			// Draw the cell as a rectangle with the specified color
			ebitenutil.DrawRect(screen, float64(x), float64(y), float64(cellXSize), float64(cellYSize), rectColor)
		}
	}

	if g.simComplete {

        ebitenutil.DebugPrintAt(screen, "Sim Complete", windowXSize/2-50, windowYSize/2)
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
	game := &Game{
		startTime: time.Now(),
	}

	// Initialize grid with random fish or empty spaces
	for i := 0; i < xdim; i++ {
		for k := 0; k < ydim; k++ {
			randomNum := rand.Intn(100) + 1 // Random number between 1 and 30
			if randomNum >= 5 && randomNum <= 10 {
				// Create and place a fish
				fish := Fish{x: i, y: k, breedTimer: 0}
				game.grid[i][k] = &fish
				game.fish = append(game.fish, fish)
			} else if randomNum >= 86 && randomNum <= 86 {
				// Create and place a shark
				shark := Shark{x: i, y: k, breedTimer: 0, starve: 0}
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

func writeSimulationDataToCSV(filename string, g *Game, threadCount int, frameRate float64) {
    // Open the CSV file in append mode (create if it doesn't exist, write-only mode)
    file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        // Log an error if the file cannot be opened
        log.Fatalf("failed to open file: %v", err)
    }
    defer file.Close() // Ensure the file is closed when the function ends

    // Create a CSV writer to write data into the file
    writer := csv.NewWriter(file)
    defer writer.Flush() // Ensure all buffered data is written to the file before the function ends

    // Get the file's stats to check if the file is empty
    stat, err := file.Stat()
    if err != nil {
        // Log an error if the file stats cannot be retrieved
        log.Fatalf("failed to get file stats: %v", err)
    }
    // If the file is empty, write the header row to the CSV file
    if stat.Size() == 0 {
        writer.Write([]string{"Grid Size", "Thread Count", "Frame Rate"})
    }

    // Prepare the data to write to the CSV file
    data := []string{
        strconv.Itoa(len(g.grid)),             // Convert the grid size to a string
        strconv.Itoa(threadCount),             // Convert the thread count to a string
        strconv.FormatFloat(frameRate, 'f', 2, 64), // Convert the frame rate to a string with 2 decimal places
    }
    // Write the prepared data to the CSV file
    if err := writer.Write(data); err != nil {
        // Log an error if the data cannot be written to the file
        log.Fatalf("failed to write to csv: %v", err)
    }
}
