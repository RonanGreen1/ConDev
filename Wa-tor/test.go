package Wator

import (
	"encoding/csv"        // Provides functions for reading and writing CSV files.
	"image/color"         // Defines colors and their manipulation for image processing.
	"log"                 // Provides logging functionality for debugging and error reporting.
	"math/rand"           // Used to generate random numbers, useful for simulation randomness.
	"os"                  // Provides functions for interacting with the operating system, such as file handling.
	"sort"                // Implements sorting algorithms for slices and user-defined collections.
	"strconv"             // Provides functions for converting strings to numbers and vice versa.
	"time"                // Provides time-related functionality, such as measuring elapsed time and delays.

	"github.com/hajimehoshi/ebiten/v2"            // A game library for building 2D games in Go.
	"github.com/hajimehoshi/ebiten/v2/ebitenutil" // Utility functions for Ebiten, such as drawing rectangles or displaying text.
)

// Constants for grid and window dimensions
const (
	xdim        = 50                // Number of cells in the x direction
	ydim        = 50                // Number of cells in the y direction
	windowXSize = 800                // Width of the window in pixels
	windowYSize = 800                // Height of the window in pixels
	cellXSize   = windowXSize / xdim // Width of each cell in pixels
	cellYSize   = windowYSize / ydim // Height of each cell in pixels
)

// Game represents the state of the simulation, including the grid and entities.
type Game struct {
	grid        [xdim][ydim]Entity // A 2D grid where each cell may contain an entity (fish, shark, or empty).
	fish        []Fish             // A slice to store all fish entities in the game.
	shark       []Shark            // A slice to store all shark entities in the game.
	startTime   time.Time          // The time when the simulation started, used for calculating metrics.
	simComplete bool               // A flag indicating whether the simulation has completed.
	totalFrames int                // Tracks the total number of frames processed during the simulation.
}

// Entity defines a common interface for all entities in the game (e.g., fish, shark).
type Entity interface {
	GetType() string            // Returns the type of the entity (e.g., "fish" or "shark").
	GetPosition() (int, int)    // Returns the current position (x, y) of the entity on the grid.
	SetPosition(x, y int)       // Updates the position of the entity on the grid.
}

// Shark represents a shark entity in the simulation.
type Shark struct {
	x, y       int // The position of the shark on the grid.
	starve     int // Tracks the number of turns since the shark last ate; used for starvation logic.
	breedTimer int // Tracks the number of turns until the shark can reproduce.
}

// GetType returns the type of the entity, which is "shark".
func (s *Shark) GetType() string {
	return "shark"
}

// GetPosition returns the current position of the shark on the grid.
func (s *Shark) GetPosition() (int, int) {
	return s.x, s.y
}

// SetPosition updates the position of the shark on the grid.
func (s *Shark) SetPosition(x, y int) {
	s.x = x
	s.y = y
}

// Fish represents a fish entity in the simulation.
type Fish struct {
	x, y       int // The position of the fish on the grid.
	breedTimer int // Tracks the number of turns until the fish can reproduce.
}

// GetType returns the type of the entity, which is "fish".
func (f *Fish) GetType() string {
	return "fish"
}

// GetPosition returns the current position of the fish on the grid.
func (f *Fish) GetPosition() (int, int) {
	return f.x, f.y
}

// SetPosition updates the position of the fish on the grid.
func (f *Fish) SetPosition(x, y int) {
	f.x = x
	f.y = y
}

// StartSimulation initializes the simulation by setting the start time and resetting the frame counter.
func (g *Game) StartSimulation() {
	g.startTime = time.Now() // Record the current time as the start of the simulation.
	g.totalFrames = 0        // Reset the total frame count to 0.
}

// RecordFrame increments the total frame count by 1.
func (g *Game) RecordFrame() {
	g.totalFrames++
}

// CalculateAverageFPS computes the average frames per second (FPS) of the simulation.
// Returns 0.0 if no time has elapsed to avoid division by zero.
func (g *Game) CalculateAverageFPS() float64 {
	elapsedTime := time.Since(g.startTime).Seconds() // Calculate elapsed time in seconds.
	if elapsedTime > 0 {
		return float64(g.totalFrames) / elapsedTime // FPS = totalFrames / elapsedTime.
	}
	return 0.0 // Default value if elapsed time is 0.
}

// Update progresses the simulation by one step.
// 
// Input:
//   - None (operates on the game state stored within the Game object).
// 
// Output:
//   - error: Returns nil unless an error occurs during the update (e.g., issues with saving results).
// 
// Functionality:
// This function handles the following tasks:
// 1. Increments the frame counter to track simulation progress.
// 2. Checks if the simulation duration exceeds 10 seconds. If so:
//    - Marks the simulation as complete.
//    - Calculates the average frames per second (FPS).
//    - Saves the results to a CSV file.
// 3. Processes fish movement and reproduction:
//    - Each fish attempts to move to a random adjacent cell.
//    - If the fish successfully moves, it increments its breeding timer.
//    - When the breeding timer reaches a threshold, the fish reproduces, creating a new fish in its previous position.
func (g *Game) Update() error {

	// RecordFrame increments the frame counter, tracking simulation progress.
	g.RecordFrame()

	// Check if the simulation duration has exceeded 10 seconds.
	if time.Since(g.startTime) > 10*time.Second {
		g.simComplete = true                      // Mark the simulation as complete.
		avgFPS := g.CalculateAverageFPS()          // Calculate the average frames per second (FPS).
		writeSimulationDataToCSV("simulation_results.csv", g, 1, avgFPS) // Save simulation results to a CSV file.
		return nil                                 // Exit the update function.
	}

	// Iterate through all fish entities to handle their movements and reproduction.
	for i := range g.fish {
		fish := &g.fish[i]         // Obtain a reference to the current fish.
		x, y := fish.GetPosition() // Get the fish's current position on the grid.

		// Attempt to move the fish in one of four random directions.
		for i := 4; i > 0; i-- {
			// Generate a random direction: 0 = north, 1 = south, 2 = east, 3 = west.
			direction := rand.Intn(i)

			newX, newY := x, y // Initialize new position with the current position.
			switch direction {
			case 0: // Move north.
				if y > 0 {
					newY = y - 1
				} else {
					newY = ydim - 1 // Wrap around to the bottom of the grid.
				}
			case 1: // Move south.
				if y < ydim-1 {
					newY = y + 1
				} else {
					newY = 0 // Wrap around to the top of the grid.
				}
			case 2: // Move east.
				if x < xdim-1 {
					newX = x + 1
				} else {
					newX = 0 // Wrap around to the left of the grid.
				}
			case 3: // Move west.
				if x > 0 {
					newX = x - 1
				} else {
					newX = xdim - 1 // Wrap around to the right of the grid.
				}
			}

			// Ensure the new position is within bounds and empty.
			if newX >= 0 && newX < xdim && newY >= 0 && newY < ydim {
				if g.grid[newX][newY] == nil { // Check if the new position is empty.
					g.grid[x][y] = nil         // Clear the fish's old position.
					fish.SetPosition(newX, newY) // Update the fish's position.
					g.grid[newX][newY] = fish  // Place the fish in its new position on the grid.

					fish.breedTimer++         // Increment the breeding timer for the fish.
					if fish.breedTimer == 5 { // Check if the fish is ready to reproduce.
						fish.breedTimer = 0    // Reset the breeding timer.
						newFish := &Fish{x: x, y: y, breedTimer: 0} // Create a new fish at the old position.
						g.grid[x][y] = newFish // Place the new fish on the grid.
						g.fish = append(g.fish, *newFish) // Add the new fish to the list of fish.
					}
					break // Exit the movement loop after successfully moving the fish.
				}
			}
		}
	}

	// Lists to track sharks and fish for removal or addition during simulation.
	removedShark := []int{}   // Indices of sharks to be removed.
	newSharks := []Shark{}    // New sharks created through reproduction.
	removedFish := []int{}    // Indices of fish to be removed.
	sharkCount := len(g.shark) // Record the initial number of sharks to prevent iteration issues.

	// Iterate through each shark to manage its behavior.
	for i := 0; i < sharkCount; i++ {
		moved := false          // Flag to indicate if the shark has moved.
		shark := &g.shark[i]    // Get a reference to the current shark.
		x, y := shark.GetPosition() // Retrieve the shark's current position.

		// Attempt to move to a cell occupied by a fish.
		for j := 0; j < 4; j++ {
			// Generate a random direction: 0 = north, 1 = south, 2 = east, 3 = west.
			direction := rand.Intn(4)

			newX, newY := x, y // Initialize new position with the current position.
			switch direction {
			case 0: // Move north.
				if y > 0 {
					newY = y - 1
				} else {
					newY = ydim - 1 // Wrap to the bottom of the grid.
				}
			case 1: // Move south.
				if y < ydim-1 {
					newY = y + 1
				} else {
					newY = 0 // Wrap to the top of the grid.
				}
			case 2: // Move east.
				if x < xdim-1 {
					newX = x + 1
				} else {
					newX = 0 // Wrap to the left of the grid.
				}
			case 3: // Move west.
				if x > 0 {
					newX = x - 1
				} else {
					newX = xdim - 1 // Wrap to the right of the grid.
				}
			}

			// Ensure the new position is within bounds and occupied by a fish.
			if newX >= 0 && newX < xdim && newY >= 0 && newY < ydim {
				if g.grid[newX][newY] != nil && g.grid[newX][newY].GetType() == "fish" {
					g.grid[x][y] = nil         // Clear the shark's old position.
					shark.SetPosition(newX, newY) // Update the shark's position.
					g.grid[newX][newY] = shark  // Place the shark in its new position.
					shark.starve = 0           // Reset the shark's starvation timer.
					shark.breedTimer++         // Increment the breeding timer for the shark.

					// Check if the shark can reproduce.
					if shark.breedTimer == 5 {
						shark.breedTimer = 0    // Reset the breeding timer.
						newShark := Shark{x: x, y: y, breedTimer: 0, starve: 0} // Create a new shark at the old position.
						g.grid[x][y] = &newShark // Place the new shark on the grid.
						newSharks = append(newSharks, newShark) // Add the new shark to the list.
					}

					// Mark the fish for removal from the grid and list.
					for j, fish := range g.fish {
						if fish.x == newX && fish.y == newY {
							removedFish = append(removedFish, j)
							break
						}
					}

					moved = true // Mark that the shark has successfully moved.
					break        // Exit the movement loop.
				}
			}
		}

		// If shark didn't move to eat a fish, attempt to move to an empty cell.
		if !moved {
			for j := 0; j < 4; j++ {
				// Generate a random direction: 0 = north, 1 = south, 2 = east, 3 = west.
				direction := rand.Intn(4)

				newX, newY := x, y // Initialize new position with the current position.
				switch direction {
				case 0: // Move north.
					if y > 0 {
						newY = y - 1
					} else {
						newY = ydim - 1 // Wrap to the bottom of the grid.
					}
				case 1: // Move south.
					if y < ydim-1 {
						newY = y + 1
					} else {
						newY = 0 // Wrap to the top of the grid.
					}
				case 2: // Move east.
					if x < xdim-1 {
						newX = x + 1
					} else {
						newX = 0 // Wrap to the left of the grid.
					}
				case 3: // Move west.
					if x > 0 {
						newX = x - 1
					} else {
						newX = xdim - 1 // Wrap to the right of the grid.
					}
				}

				// Ensure the new position is within bounds and empty.
				if newX >= 0 && newX < xdim && newY >= 0 && newY < ydim {
					if g.grid[newX][newY] == nil { // Check if the new position is empty.
						g.grid[x][y] = nil         // Clear the shark's old position.
						shark.SetPosition(newX, newY) // Update the shark's position.
						g.grid[newX][newY] = shark  // Place the shark in its new position on the grid.

						shark.starve++             // Increment the shark's starvation timer.
						if shark.starve == 5 {     // Check if the shark has starved.
							g.grid[newX][newY] = nil // Remove the shark from the grid.
							removedShark = append(removedShark, i) // Mark the shark for removal.
						}

						shark.breedTimer++         // Increment the breeding timer for the shark.
						if shark.breedTimer == 6 { // Check if the shark can reproduce.
							shark.breedTimer = 0    // Reset the breeding timer.
							newShark := Shark{x: x, y: y, breedTimer: 0, starve: 0} // Create a new shark at the old position.
							g.grid[x][y] = &newShark // Place the new shark on the grid.
							newSharks = append(newSharks, newShark) // Add the new shark to the list.
						}

						moved = true // Mark that the shark has successfully moved.
						break        // Exit the movement loop.
					}
				}
			}
		}
	}
	// Remove fish that were eaten.
	sort.Sort(sort.Reverse(sort.IntSlice(removedFish))) // Sort in reverse order to remove elements from the end first, avoiding index shift issues.
	for _, index := range removedFish {                 // Iterate over the sorted indices and remove the fish.
		if index < len(g.fish) {
			g.fish = append(g.fish[:index], g.fish[index+1:]...) // Remove the fish from the list.
		}
	}

	// Remove sharks that starved.
	sort.Sort(sort.Reverse(sort.IntSlice(removedShark))) // Sort in reverse order to remove elements from the end first, avoiding index shift issues.
	for _, index := range removedShark {                 // Iterate over the sorted indices and remove the sharks.
		if index < len(g.shark) {
			g.shark = append(g.shark[:index], g.shark[index+1:]...) // Remove the shark from the list.
		}
	}

	// Add new sharks after iteration.
	g.shark = append(g.shark, newSharks...) // Append newly created sharks to the list.

	return nil // Return nil to indicate the update completed successfully.
}


// Draw renders the game grid and entities to the screen.
// 
// Input:
//   - screen (*ebiten.Image): The screen object where the game grid and entities will be drawn.
// 
// Output:
//   - None (updates the screen object directly).
// 
// Functionality:
// This function updates the game display by iterating over the game grid and rendering each cell with a color corresponding to its content.
// - "fish" entities are drawn as light blue rectangles.
// - "shark" entities are drawn as purple rectangles.
// - Empty cells are transparent.
// Additionally, if the simulation is marked as complete, a completion message is displayed at the center of the screen.
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black) // Clear the screen with black color.

	// Iterate over each cell in the grid.
	for i := 0; i < xdim; i++ {
		for k := 0; k < ydim; k++ {
			// Calculate the position of the current cell in pixels.
			x := i * cellXSize
			y := k * cellYSize

			// Determine the color based on the entity in the cell.
			var rectColor color.Color
			if entity := g.grid[i][k]; entity != nil {
				switch entity.GetType() {
				case "fish":
					rectColor = color.RGBA{0, 221, 255, 1} // Light blue for fish.
				case "shark":
					rectColor = color.RGBA{190, 44, 190, 1} // Purple for shark.
				}
			} else {
				rectColor = color.RGBA{0, 0, 0, 0} // Transparent for empty cells.
			}

			// Draw the cell as a rectangle with the specified color.
			ebitenutil.DrawRect(screen, float64(x), float64(y), float64(cellXSize), float64(cellYSize), rectColor)
		}
	}

	// If the simulation is complete, display a completion message.
	if g.simComplete {
		ebitenutil.DebugPrintAt(screen, "Sim Complete", windowXSize/2-50, windowYSize/2) // Center the message.
	}
}


// Layout sets the dimensions of the game window.
// 
// Input:
//   - outsideWidth (int): The external width of the window, passed by the game engine.
//   - outsideHeight (int): The external height of the window, passed by the game engine.
// 
// Output:
//   - (int, int): The internal width and height of the game window, which remain constant.
// 
// Functionality:
// This function ensures that the game's window dimensions are consistent regardless of external inputs.
// It is called by the Ebiten game engine to determine the size of the game's rendering area.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return windowXSize, windowYSize
}

// NewGame initializes a new game instance with a grid of cells and random entities (fish, sharks, or empty spaces).
// 
// Input:
//   - None.
// 
// Output:
//   - *Game: A pointer to the newly created Game instance.
// 
// Functionality:
// This function sets up the initial state of the game, including the grid, fish, and sharks:
// - A 2D grid of dimensions `xdim` by `ydim` is created.
// - Each cell in the grid is randomly assigned to contain a fish, a shark, or remain empty based on a random number.
// - Fish and sharks are initialized with default properties, such as their position and timers.
// 
// Details:
// - Fish occupy cells with a random number between 5 and 10 (inclusive).
// - Sharks occupy cells with a specific random number (e.g., 86).
// - Other cells are left empty.
func NewGame() *Game {
	game := &Game{
		startTime: time.Now(), // Record the start time of the game.
	}

	// Initialize grid with random entities.
	for i := 0; i < xdim; i++ {
		for k := 0; k < ydim; k++ {
			randomNum := rand.Intn(100) + 1 // Generate a random number between 1 and 100.
			if randomNum >= 5 && randomNum <= 10 {
				// Create and place a fish in the current cell.
				fish := Fish{x: i, y: k, breedTimer: 0}
				game.grid[i][k] = &fish
				game.fish = append(game.fish, fish) // Add the fish to the list of all fish.
			} else if randomNum == 86 {
				// Create and place a shark in the current cell.
				shark := Shark{x: i, y: k, breedTimer: 0, starve: 0}
				game.grid[i][k] = &shark
				game.shark = append(game.shark, shark) // Add the shark to the list of all sharks.
			} else {
				// Leave the cell empty.
				game.grid[i][k] = nil
			}
		}
	}

	return game // Return the newly created Game instance.
}

// main is the entry point of the program.
// 
// Input:
//   - None (execution starts from the main function).
// 
// Output:
//   - None (executes the game loop or logs an error on failure).
// 
// Functionality:
// The main function initializes and starts the simulation:
// 1. Calls NewGame to create a new game instance, which sets up the initial grid and entities.
// 2. Configures the game window by setting its size and title using Ebiten's functions.
// 3. Starts the game loop using `ebiten.RunGame`:
//    - Ebiten repeatedly calls the Update and Draw methods of the Game instance.
//    - The simulation runs until manually terminated or an error occurs.
// 4. If an error occurs during the game loop, it is logged and the program exits.
func main() {
	game := NewGame() // Create a new game instance.

	// Set the window size and title for the simulation.
	ebiten.SetWindowSize(windowXSize, windowYSize)       // Define the window dimensions.
	ebiten.SetWindowTitle("Ebiten Wa-Tor World")        // Set the window title.

	// Run the game loop, which continuously updates and draws the game state.
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err) // Log any errors that occur and terminate the program.
	}
}

// writeSimulationDataToCSV writes simulation performance data to a CSV file.
// 
// Input:
//   - filename (string): The name of the CSV file where data will be written.
//   - g (*Game): The current game instance containing the simulation's state.
//   - threadCount (int): The number of threads used in the simulation.
//   - frameRate (float64): The average frame rate during the simulation.
// 
// Output:
//   - None (writes data to a file or terminates the program on error).
// 
// Functionality:
// This function appends simulation data to a CSV file, creating the file if it does not already exist:
// 1. Opens the file in append mode (or creates it if it doesn't exist).
// 2. Ensures the file has the appropriate header row if it's empty.
// 3. Converts simulation data (grid size, thread count, frame rate) to strings and writes them as a row in the CSV file.
// 4. Logs and terminates the program if any file operation fails.
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
	    strconv.Itoa(xdim * ydim),             // Convert the grid size to a string
	    strconv.Itoa(threadCount),             // Convert the thread count to a string
	    strconv.FormatFloat(frameRate, 'f', 2, 64), // Convert the frame rate to a string with 2 decimal places
	}
	// Write the prepared data to the CSV file
	if err := writer.Write(data); err != nil {
	// Log an error if the data cannot be written to the file
	    log.Fatalf("failed to write to csv: %v", err)
	}
}
