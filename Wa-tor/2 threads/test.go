package main

import (
    "encoding/csv"               // Package for reading and writing CSV files.
    "image/color"                // Package for handling colors used in rendering.
    "log"                        // Package for logging errors and information.
    "math/rand"                  // Package for generating random numbers.
    "os"                         // Package for interacting with the operating system (e.g., file handling).
    "strconv"                    // Package for converting data types to and from strings.
    "sync"                       // Package for handling synchronization (e.g., mutexes for safe concurrent access).
    "time"                       // Package for handling time and duration.

    "github.com/hajimehoshi/ebiten/v2"             // Ebiten package for creating 2D games.
    "github.com/hajimehoshi/ebiten/v2/ebitenutil"  // Utility functions for Ebiten, such as drawing shapes and debugging.
)

// Constants for grid and window dimensions.
const (
    xdim        = 50                 // Number of cells in the x direction (grid width).
    ydim        = 50                 // Number of cells in the y direction (grid height).
    windowXSize = 800                 // Width of the game window in pixels.
    windowYSize = 800                 // Height of the game window in pixels.
    cellXSize   = windowXSize / xdim  // Width of each cell in pixels, calculated based on the grid and window size.
    cellYSize   = windowYSize / ydim  // Height of each cell in pixels, calculated similarly.
)

// Game struct representing the state of the game.
// Contains the grid, entities (fish and sharks), simulation metadata, and synchronization primitives.
type Game struct {
    grid        [xdim][ydim]Entity  // 2D array representing the game grid; each cell holds an Entity (fish, shark, or nil).
    fish        []*Fish             // List of all fish in the simulation.
    shark       []*Shark            // List of all sharks in the simulation.
    startTime   time.Time           // Time when the simulation started.
    simComplete bool                // Flag indicating whether the simulation is complete.
    totalFrames int                 // Counter for the total number of frames rendered.
    partitions  []Partition         // List of partitions dividing the grid for multithreaded processing.
    fishMutex   sync.Mutex          // Mutex for safely modifying the fish list.
    sharkMutex  sync.Mutex          // Mutex for safely modifying the shark list.
}

// Partition struct representing a section of the grid.
// Used to divide the grid for concurrent processing.
type Partition struct {
    startX             int          // Starting x-coordinate of the partition.
    endX               int          // Ending x-coordinate of the partition.
    leftBoundaryMutex  *sync.Mutex  // Mutex for controlling access to the left boundary of the partition.
    rightBoundaryMutex *sync.Mutex  // Mutex for controlling access to the right boundary of the partition.
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

// Update updates the game state every frame.
// 
// Input:
//   - None (operates on the game state stored within the Game object).
// 
// Output:
//   - error: Returns nil unless an error occurs during the update (e.g., issues with saving results).
// 
// Functionality:
// This function handles the simulation logic, including:
// 1. Recording each frame to track simulation progress.
// 2. Checking if the simulation has exceeded its time limit (10 seconds):
//    - If complete, calculates the average FPS and writes the results to a CSV file.
// 3. Dividing the grid into partitions for concurrent updates using goroutines.
//    - Each partition processes entities within its bounds.
// 4. Waiting for all partitions to finish using a `sync.WaitGroup`.
// 5. Consolidating updates to the game state after all partitions are processed.
func (g *Game) Update() error {
    g.RecordFrame() // Record the current frame count for performance tracking.

    // Check if the simulation duration has exceeded 10 seconds.
    if time.Since(g.startTime) > 10*time.Second {
        g.simComplete = true // Mark the simulation as complete.
        avgFPS := g.CalculateAverageFPS() // Calculate the average FPS.
        // Save the simulation results to a CSV file.
        writeSimulationDataToCSV("simulation_results_2_threads.csv", g, len(g.partitions), avgFPS)
        return nil // Exit the update function as the simulation is complete.
    }

    var wg sync.WaitGroup             // Create a WaitGroup to synchronize goroutines.
    wg.Add(len(g.partitions))         // Add the number of partitions to the WaitGroup counter.

    // Prepare slices to collect results for fish and sharks.
    allFishAdditions := make([][]*Fish, len(g.partitions))  // Slices to collect fish added in each partition.
    allFishRemovals := make([][]*Fish, len(g.partitions))   // Slices to collect fish removed in each partition.
    allSharkAdditions := make([][]*Shark, len(g.partitions))// Slices to collect sharks added in each partition.
    allSharkRemovals := make([][]*Shark, len(g.partitions)) // Slices to collect sharks removed in each partition.

    // Iterate over each partition and process it concurrently.
    for i, partition := range g.partitions {
        go func(i int, p Partition) {
            defer wg.Done() // Decrement the WaitGroup counter when the goroutine finishes.
            // Run the simulation logic for this partition and collect results.
            fa, fr, sa, sr := g.RunPartition(p)
            allFishAdditions[i] = fa // Store fish additions for this partition.
            allFishRemovals[i] = fr  // Store fish removals for this partition.
            allSharkAdditions[i] = sa// Store shark additions for this partition.
            allSharkRemovals[i] = sr // Store shark removals for this partition.
        }(i, partition) // Pass the partition and its index to the goroutine.
    }

    wg.Wait() // Wait for all partition goroutines to finish execution.

    // Process all additions and removals collected from the partitions.
    g.processRemovalsAndAdditions(allFishAdditions, allFishRemovals, allSharkAdditions, allSharkRemovals)

    return nil // Return nil to indicate the update completed successfully.
}

// processRemovalsAndAdditions consolidates and updates the game state by handling additions and removals of fish and sharks.
// 
// Input:
//   - allFishAdditions ([][]*Fish): A collection of fish additions from all partitions.
//   - allFishRemovals ([][]*Fish): A collection of fish removals from all partitions.
//   - allSharkAdditions ([][]*Shark): A collection of shark additions from all partitions.
//   - allSharkRemovals ([][]*Shark): A collection of shark removals from all partitions.
// 
// Output:
//   - None (modifies the game state directly).
// 
// Functionality:
// 1. Combines all additions and removals from partitions into single slices.
// 2. Updates the game's list of fish and sharks by removing specified entities and appending new ones.
// 3. Uses mutex locks to ensure thread-safe updates to shared resources.
func (g *Game) processRemovalsAndAdditions(
    allFishAdditions [][]*Fish, allFishRemovals [][]*Fish,
    allSharkAdditions [][]*Shark, allSharkRemovals [][]*Shark) {

    var fishAdditions []*Fish
    var fishRemovals []*Fish
    var sharkAdditions []*Shark
    var sharkRemovals []*Shark

    // Combine slices of fish additions from all partitions.
    for _, fa := range allFishAdditions {
        fishAdditions = append(fishAdditions, fa...) // Append each partition's additions to the main slice.
    }

    // Combine slices of fish removals from all partitions.
    for _, fr := range allFishRemovals {
        fishRemovals = append(fishRemovals, fr...) // Append each partition's removals to the main slice.
    }

    // Combine slices of shark additions from all partitions.
    for _, sa := range allSharkAdditions {
        sharkAdditions = append(sharkAdditions, sa...) // Append each partition's additions to the main slice.
    }

    // Combine slices of shark removals from all partitions.
    for _, sr := range allSharkRemovals {
        sharkRemovals = append(sharkRemovals, sr...) // Append each partition's removals to the main slice.
    }

    // Remove fish marked for removal.
    fishToRemove := make(map[*Fish]bool) // Create a map to mark fish for removal.
    for _, fish := range fishRemovals {
        fishToRemove[fish] = true
    }

    g.fishMutex.Lock() // Lock the fish mutex to ensure thread-safe access.
    var newFish []*Fish
    for _, fish := range g.fish {
        if !fishToRemove[fish] { // Retain fish not marked for removal.
            newFish = append(newFish, fish)
        }
    }
    g.fish = newFish                     // Update the fish list with retained fish.
    g.fish = append(g.fish, fishAdditions...) // Append newly added fish.
    g.fishMutex.Unlock() // Unlock the fish mutex.

    // Remove sharks marked for removal.
    sharkToRemove := make(map[*Shark]bool) // Create a map to mark sharks for removal.
    for _, shark := range sharkRemovals {
        sharkToRemove[shark] = true
    }

    g.sharkMutex.Lock() // Lock the shark mutex to ensure thread-safe access.
    var newSharks []*Shark
    for _, shark := range g.shark {
        if !sharkToRemove[shark] { // Retain sharks not marked for removal.
            newSharks = append(newSharks, shark)
        }
    }
    g.shark = newSharks                     // Update the shark list with retained sharks.
    g.shark = append(g.shark, sharkAdditions...) // Append newly added sharks.
    g.sharkMutex.Unlock() // Unlock the shark mutex.
}

// RunPartition processes a specific partition of the grid for fish and shark movements and updates.
// 
// Input:
//   - p (Partition): A section of the grid defined by start and end x-coordinates and associated boundary mutexes.
// 
// Output:
//   - ([]*Fish, []*Fish, []*Shark, []*Shark):
//       - A slice of new fish added within the partition.
//       - A slice of fish to be removed from the partition.
//       - A slice of new sharks added within the partition.
//       - A slice of sharks to be removed from the partition.
// 
// Functionality:
// 1. Copies the current lists of fish and sharks to avoid concurrent access issues.
// 2. Processes each fish within the partition, attempting to:
//    - Move it to a new cell.
//    - Breed a new fish if the breed timer threshold is reached.
// 3. Ensures thread safety when crossing partition boundaries by locking and unlocking boundary mutexes.
func (g *Game) RunPartition(p Partition) ([]*Fish, []*Fish, []*Shark, []*Shark) {
    // Local slices for tracking additions and removals of fish and sharks in this partition.
    var localFishAdditions []*Fish
    var localFishRemovals []*Fish
    var localSharkAdditions []*Shark
    var localSharkRemovals []*Shark

    // Create a copy of the fish list to avoid concurrent read issues.
    g.fishMutex.Lock()                     // Lock the fish mutex to safely access the shared list.
    fishCopy := make([]*Fish, len(g.fish)) // Create a slice for the fish copy.
    copy(fishCopy, g.fish)                 // Copy the shared fish slice into the local slice.
    g.fishMutex.Unlock()                   // Unlock the fish mutex.

    // Create a copy of the shark list to avoid concurrent read issues.
    g.sharkMutex.Lock()                      // Lock the shark mutex to safely access the shared list.
    sharkCopy := make([]*Shark, len(g.shark)) // Create a slice for the shark copy.
    copy(sharkCopy, g.shark)                 // Copy the shared shark slice into the local slice.
    g.sharkMutex.Unlock()                    // Unlock the shark mutex.

    // Process each fish in the copied list.
    for _, fish := range fishCopy {
        x, y := fish.GetPosition() // Get the current position of the fish.

        // Skip fish that are outside the partition boundaries.
        if x < p.startX || x > p.endX {
            continue
        }

        moved := false // Flag to track if the fish has moved.

        // Attempt to move the fish in up to four random directions.
        for dir := 0; dir < 4; dir++ {
            direction := rand.Intn(4) // Randomly select a direction (0 = north, 1 = south, etc.).

            newX, newY := x, y // Initialize new position variables.

            // Determine the new position based on the direction.
            switch direction {
            case 0: // Move north.
                if y > 0 {
                    newY = y - 1
                } else {
                    newY = ydim - 1 // Wrap around to the bottom.
                }
            case 1: // Move south.
                if y < ydim-1 {
                    newY = y + 1
                } else {
                    newY = 0 // Wrap around to the top.
                }
            case 2: // Move east.
                if x < xdim-1 {
                    newX = x + 1
                } else {
                    newX = 0 // Wrap around to the left.
                }
            case 3: // Move west.
                if x > 0 {
                    newX = x - 1
                } else {
                    newX = xdim - 1 // Wrap around to the right.
                }
            }

            // Variable to hold the mutex if crossing a boundary.
            var mu *sync.Mutex

            // Check if the new position crosses a partition boundary.
            if newX < p.startX {
                mu = p.leftBoundaryMutex // Use the left boundary mutex.
                mu.Lock()                // Lock the left boundary mutex.
            } else if newX > p.endX {
                mu = p.rightBoundaryMutex // Use the right boundary mutex.
                mu.Lock()                 // Lock the right boundary mutex.
            }

            // Check if the new cell is empty.
            if g.grid[newX][newY] == nil {
                g.grid[x][y] = nil           // Clear the fish's current cell.
                fish.SetPosition(newX, newY) // Update the fish's position.
                g.grid[newX][newY] = fish    // Place the fish in the new cell.

                fish.breedTimer++ // Increment the fish's breed timer.

                // Check if the fish is ready to breed.
                if fish.breedTimer == 5 {
                    fish.breedTimer = 0 // Reset the breed timer.
                    // Create a new fish at the old position.
                    newFish := &Fish{x: x, y: y, breedTimer: 0}
                    g.grid[x][y] = newFish                     // Place the new fish in the old cell.
                    localFishAdditions = append(localFishAdditions, newFish) // Add the new fish to local additions.
                }

                moved = true // Mark that the fish has moved.
            }

            // Unlock the boundary mutex if it was used.
            if mu != nil {
                mu.Unlock()
            }

            // Exit the loop if the fish has successfully moved.
            if moved {
                break
            }
        }
    }

    for _, shark := range sharkCopy {
        x, y := shark.GetPosition() // Get the current position of the shark.
    
        // Skip sharks that are outside the partition boundaries.
        if x < p.startX || x > p.endX {
            continue
        }
    
        moved := false // Flag to track if the shark has moved.
    
        // Attempt to move the shark up to four times in a random direction.
        for dir := 0; dir < 4; dir++ {
            direction := rand.Intn(4) // Randomly select a direction (0 = north, 1 = south, etc.).
    
            newX, newY := x, y // Initialize new position variables.
    
            // Determine the new position based on the direction.
            switch direction {
            case 0: // Move north.
                if y > 0 {
                    newY = y - 1
                } else {
                    newY = ydim - 1 // Wrap around to the bottom.
                }
            case 1: // Move south.
                if y < ydim-1 {
                    newY = y + 1
                } else {
                    newY = 0 // Wrap around to the top.
                }
            case 2: // Move east.
                if x < xdim-1 {
                    newX = x + 1
                } else {
                    newX = 0 // Wrap around to the left.
                }
            case 3: // Move west.
                if x > 0 {
                    newX = x - 1
                } else {
                    newX = xdim - 1 // Wrap around to the right.
                }
            }
    
            // Variable to hold the boundary mutex if crossing a boundary.
            var mu *sync.Mutex
    
            // Check if the new position crosses a partition boundary.
            if newX < p.startX {
                mu = p.leftBoundaryMutex // Use the left boundary mutex.
                mu.Lock()                // Lock the left boundary mutex.
            } else if newX > p.endX {
                mu = p.rightBoundaryMutex // Use the right boundary mutex.
                mu.Lock()                 // Lock the right boundary mutex.
            }
    
            // Check if the new cell is occupied by a fish.
            if g.grid[newX][newY] != nil && g.grid[newX][newY].GetType() == "fish" {
                g.grid[x][y] = nil            // Clear the shark's current cell.
                shark.SetPosition(newX, newY) // Update the shark's position.
                g.grid[newX][newY] = shark    // Place the shark in the new cell.
    
                shark.starve = 0 // Reset the shark's starvation counter.
    
                // Increment the shark's breed timer.
                shark.breedTimer++
                if shark.breedTimer == 5 {
                    shark.breedTimer = 0 // Reset the breed timer.
                    // Create a new shark at the old position.
                    newShark := &Shark{x: x, y: y, breedTimer: 0, starve: 0}
                    g.grid[x][y] = newShark                       // Place the new shark in the old cell.
                    localSharkAdditions = append(localSharkAdditions, newShark) // Add the new shark to local additions.
                }
    
                // Mark the fish for removal from the fish slice.
                var fishToRemove *Fish
                for _, fish := range fishCopy {
                    fx, fy := fish.GetPosition() // Get the fish's position.
                    if fx == newX && fy == newY {
                        fishToRemove = fish // Identify the fish to remove.
                        break
                    }
                }
                if fishToRemove != nil {
                    localFishRemovals = append(localFishRemovals, fishToRemove) // Add the fish to local removals.
                }
    
                moved = true // Mark that the shark has moved.
            }
    
            // Unlock the boundary mutex if it was used.
            if mu != nil {
                mu.Unlock()
            }
    
            // Exit the loop if the shark has successfully moved.
            if moved {
                break
            }
        }

        if !moved { // If the shark didn't move by eating a fish.
            for dir := 0; dir < 4; dir++ {
                direction := rand.Intn(4) // Randomly select a direction (0 = north, 1 = south, etc.).
        
                newX, newY := x, y // Initialize new position variables.
        
                // Determine the new position based on the direction.
                switch direction {
                case 0: // Move north.
                    if y > 0 {
                        newY = y - 1
                    } else {
                        newY = ydim - 1 // Wrap around to the bottom.
                    }
                case 1: // Move south.
                    if y < ydim-1 {
                        newY = y + 1
                    } else {
                        newY = 0 // Wrap around to the top.
                    }
                case 2: // Move east.
                    if x < xdim-1 {
                        newX = x + 1
                    } else {
                        newX = 0 // Wrap around to the left.
                    }
                case 3: // Move west.
                    if x > 0 {
                        newX = x - 1
                    } else {
                        newX = xdim - 1 // Wrap around to the right.
                    }
                }
        
                // Variable to hold the boundary mutex if crossing a boundary.
                var mu *sync.Mutex
        
                // Check if the new position crosses a partition boundary.
                if newX < p.startX {
                    mu = p.leftBoundaryMutex // Use the left boundary mutex.
                    mu.Lock()                // Lock the left boundary mutex.
                } else if newX > p.endX {
                    mu = p.rightBoundaryMutex // Use the right boundary mutex.
                    mu.Lock()                 // Lock the right boundary mutex.
                }
        
                // Check if the new cell is empty.
                if g.grid[newX][newY] == nil {
                    g.grid[x][y] = nil            // Clear the current cell.
                    shark.SetPosition(newX, newY) // Update the shark's position.
                    g.grid[newX][newY] = shark    // Place the shark in the new cell.
        
                    shark.starve++ // Increment the shark's starvation counter.
        
                    // Check if the shark has died of starvation.
                    if shark.starve == 5 {
                        g.grid[newX][newY] = nil                     // Remove the shark from the grid.
                        localSharkRemovals = append(localSharkRemovals, shark) // Mark the shark for removal.
                    } else {
                        // Increment the shark's breeding timer.
                        shark.breedTimer++
                        if shark.breedTimer == 6 {
                            shark.breedTimer = 0 // Reset the breeding timer.
                            // Create a new shark at the old position.
                            newShark := &Shark{x: x, y: y, breedTimer: 0, starve: 0}
                            g.grid[x][y] = newShark                       // Place the new shark in the old cell.
                            localSharkAdditions = append(localSharkAdditions, newShark) // Add the new shark to local additions.
                        }
                    }
        
                    moved = true // Mark that the shark has moved.
                }
        
                // Unlock the boundary mutex if it was used.
                if mu != nil {
                    mu.Unlock()
                }
        
                // Exit the loop if the shark has successfully moved.
                if moved {
                    break
                }
            }
        }
    }

    // Return local additions and removals
    return localFishAdditions, localFishRemovals, localSharkAdditions, localSharkRemovals
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

// NewGame initializes a new game instance with a grid of cells and partitioning for multi-threading.
//
// Input:
//   - None.
//
// Output:
//   - *Game: A pointer to the newly initialized Game instance.
//
// Functionality:
// 1. Creates a game instance and sets the start time.
// 2. Divides the grid into two partitions for two threads.
//    - Each partition has boundary mutexes for thread-safe updates at edges.
// 3. Initializes the grid with random entities (fish, sharks, or empty spaces).
//    - Fish and sharks are placed with specified probabilities.
//    - Populates the fish and shark lists with their respective entities.
func NewGame() *Game {
    // Initialize a new Game instance with the current start time.
    game := &Game{
        startTime: time.Now(),
    }

    // Divide the grid into two partitions for multi-threading.
    partitionSize := xdim / 2 // Half the grid width for two threads.

    // Create mutexes for managing boundary synchronization.
    leftBoundaryMutex := &sync.Mutex{}  // Mutex for the left boundary.
    rightBoundaryMutex := &sync.Mutex{} // Mutex for the right boundary.

    // Define partitions for the grid, ensuring mutexes are shared appropriately.
    game.partitions = []Partition{
        {
            startX:             0,                    // Start of the first partition.
            endX:               partitionSize - 1,    // End of the first partition.
            leftBoundaryMutex:  leftBoundaryMutex,    // Mutex for the left boundary.
            rightBoundaryMutex: rightBoundaryMutex,   // Mutex for the right boundary.
        },
        {
            startX:             partitionSize,        // Start of the second partition.
            endX:               xdim - 1,             // End of the second partition.
            leftBoundaryMutex:  rightBoundaryMutex,   // Mutex for the shared boundary.
            rightBoundaryMutex: leftBoundaryMutex,    // Mutex for the other shared boundary.
        },
    }

    // Populate the grid with random entities.
    for i := 0; i < xdim; i++ {        // Iterate over the x-dimension.
        for k := 0; k < ydim; k++ {    // Iterate over the y-dimension.
            randomNum := rand.Intn(100) + 1 // Generate a random number between 1 and 100.

            if randomNum >= 5 && randomNum <= 10 { // 6% chance to place a fish.
                fish := &Fish{x: i, y: k, breedTimer: 0} // Create a new fish entity.
                game.grid[i][k] = fish                  // Place the fish on the grid.
                game.fish = append(game.fish, fish)     // Add the fish to the game's fish list.
            } else if randomNum == 86 { // 1% chance to place a shark.
                shark := &Shark{x: i, y: k, breedTimer: 0, starve: 0} // Create a new shark entity.
                game.grid[i][k] = shark                               // Place the shark on the grid.
                game.shark = append(game.shark, shark)                // Add the shark to the game's shark list.
            } else {
                game.grid[i][k] = nil // Leave the cell empty.
            }
        }
    }

    return game // Return the newly created game instance.
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