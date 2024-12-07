package main

import (
    "encoding/csv"          // Handles reading and writing CSV files, used for logging simulation data.
    "image/color"           // Provides color definitions and manipulations, used for visualising the simulation grid.
    "log"                   // For logging errors or other significant events during runtime.
    "math/rand"             // Generates random numbers, used for fish and shark movement and population initialisation.
    "os"                    // Handles file operations, such as opening, writing, or appending data to CSV files.
    "sort"                  // Offers utilities for sorting slices, used for ordering mutexes or other collections.
    "sync"                  // Provides concurrency primitives like Mutex and WaitGroup for thread-safe operations.
    "time"                  // Provides utilities for working with time, such as timers or calculating simulation duration.
    "unsafe"                // Enables low-level operations, used for pointer-based sorting in mutexes.
    "strconv"               // Converts strings to other types and vice versa, such as for CSV data formatting.

	"github.com/hajimehoshi/ebiten/v2"            // A game library for building 2D games in Go.
	"github.com/hajimehoshi/ebiten/v2/ebitenutil" // Utility functions for Ebiten, such as drawing rectangles or displaying text.
)

// Constants for grid and window dimensions
const (
    xdim        = 50                 // Number of cells in the x direction
    ydim        = 50                 // Number of cells in the y direction
    windowXSize = 800                // Width of the window in pixels
    windowYSize = 800                // Height of the window in pixels
    cellXSize   = windowXSize / xdim // Width of each cell in pixels
    cellYSize   = windowYSize / ydim // Height of each cell in pixels
)

// Game struct representing the state of the game
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

// Partition struct representing a section of the grid
type Partition struct {
    startX int
    endX   int
    startY int
    endY   int

    // Boundary mutexes for synchronization
    leftBoundaryMutex   *sync.Mutex
    rightBoundaryMutex  *sync.Mutex
    topBoundaryMutex    *sync.Mutex
    bottomBoundaryMutex *sync.Mutex
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
        //avgFPS := g.CalculateAverageFPS() // Calculate the average FPS.
        // Save the simulation results to a CSV file.
        //writeSimulationDataToCSV("simulation_results_2_threads.csv", g, len(g.partitions), avgFPS)
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
    // Local slices for additions and removals of fish and sharks.
    var localFishAdditions []*Fish
    var localFishRemovals []*Fish
    var localSharkAdditions []*Shark
    var localSharkRemovals []*Shark

    // Create a copy of g.fish to avoid concurrent read issues.
    g.fishMutex.Lock()
    fishCopy := make([]*Fish, len(g.fish))
    copy(fishCopy, g.fish)
    g.fishMutex.Unlock()

    // Create a copy of g.shark to avoid concurrent read issues.
    g.sharkMutex.Lock()
    sharkCopy := make([]*Shark, len(g.shark))
    copy(sharkCopy, g.shark)
    g.sharkMutex.Unlock()

    // Process each fish in the copied fish slice.
    for _, fish := range fishCopy {
        x, y := fish.GetPosition() // Get the current position of the fish.

        // Check if the fish is within this partition.
        if x < p.startX || x > p.endX || y < p.startY || y > p.endY {
            continue // Skip fish not in this partition.
        }

        moved := false // Flag to track if the fish has moved.

        // Try moving the fish in up to four random directions.
        for dir := 0; dir < 4; dir++ {
            direction := rand.Intn(4) // Randomly select a direction (0-3).

            newX, newY := x, y // Initialize the new position variables.

            // Determine the new position based on the direction.
            switch direction {
            case 0: // North.
                if y > 0 {
                    newY = y - 1
                } else {
                    newY = ydim - 1 // Wrap around to the bottom.
                }
            case 1: // South.
                if y < ydim-1 {
                    newY = y + 1
                } else {
                    newY = 0 // Wrap around to the top.
                }
            case 2: // East.
                if x < xdim-1 {
                    newX = x + 1
                } else {
                    newX = 0 // Wrap around to the left.
                }
            case 3: // West.
                if x > 0 {
                    newX = x - 1
                } else {
                    newX = xdim - 1 // Wrap around to the right.
                }
            }

            // Determine if the movement crosses boundaries.
            var boundaryMutexes []*sync.Mutex

            if (x == p.startX && newX < x) || (x == p.endX && newX > x) {
                // Crosses a vertical boundary.
                if newX < x && p.leftBoundaryMutex != nil {
                    boundaryMutexes = append(boundaryMutexes, p.leftBoundaryMutex)
                }
                if newX > x && p.rightBoundaryMutex != nil {
                    boundaryMutexes = append(boundaryMutexes, p.rightBoundaryMutex)
                }
            }

            if (y == p.startY && newY < y) || (y == p.endY && newY > y) {
                // Crosses a horizontal boundary.
                if newY < y && p.topBoundaryMutex != nil {
                    boundaryMutexes = append(boundaryMutexes, p.topBoundaryMutex)
                }
                if newY > y && p.bottomBoundaryMutex != nil {
                    boundaryMutexes = append(boundaryMutexes, p.bottomBoundaryMutex)
                }
            }

            // Sort and lock boundary mutexes to ensure consistent locking order.
            sort.Slice(boundaryMutexes, func(i, j int) bool {
                return uintptr(unsafe.Pointer(boundaryMutexes[i])) < uintptr(unsafe.Pointer(boundaryMutexes[j]))
            })
            for _, mu := range boundaryMutexes {
                mu.Lock()
            }

            // Check if the new cell is empty.
            if g.grid[newX][newY] == nil {
                // Move the fish to the new position.
                g.grid[x][y] = nil           // Clear the current cell.
                fish.SetPosition(newX, newY) // Update fish's position.
                g.grid[newX][newY] = fish    // Place fish in the new cell.

                // Increment the fish's breed timer.
                fish.breedTimer++
                if fish.breedTimer == 5 {
                    // Fish is ready to breed.
                    fish.breedTimer = 0
                    // Create a new fish at the old position.
                    newFish := &Fish{x: x, y: y, breedTimer: 0}
                    g.grid[x][y] = newFish                    // Place the new fish in the old cell.
                    localFishAdditions = append(localFishAdditions, newFish) // Add to local additions.
                }
                moved = true // Mark that the fish has moved.
            }

            // Unlock boundary mutexes in reverse order.
            for i := len(boundaryMutexes) - 1; i >= 0; i-- {
                boundaryMutexes[i].Unlock()
            }

            if moved {
                break // Exit the direction loop if the fish has moved.
            }
        }
    }

    for _, shark := range sharkCopy {
        x, y := shark.GetPosition() // Get the current position of the shark.
    
        // Check if the shark is within this partition.
        if x < p.startX || x > p.endX || y < p.startY || y > p.endY {
            continue // Skip sharks not in this partition.
        }
    
        moved := false // Flag to track if the shark has moved.
    
        // Try to move to a position occupied by a fish first.
        for dir := 0; dir < 4; dir++ {
            direction := rand.Intn(4) // Randomly select a direction (0-3).
    
            newX, newY := x, y // Initialize the new position variables.
    
            // Determine the new position based on the direction.
            switch direction {
            case 0: // North.
                if y > 0 {
                    newY = y - 1
                } else {
                    newY = ydim - 1 // Wrap around to the bottom.
                }
            case 1: // South.
                if y < ydim-1 {
                    newY = y + 1
                } else {
                    newY = 0 // Wrap around to the top.
                }
            case 2: // East.
                if x < xdim-1 {
                    newX = x + 1
                } else {
                    newX = 0 // Wrap around to the left.
                }
            case 3: // West.
                if x > 0 {
                    newX = x - 1
                } else {
                    newX = xdim - 1 // Wrap around to the right.
                }
            }
    
            // Determine if the movement crosses boundaries.
            var boundaryMutexes []*sync.Mutex
    
            if (x == p.startX && newX < x) || (x == p.endX && newX > x) {
                // Crossing vertical boundary.
                if newX < x && p.leftBoundaryMutex != nil {
                    boundaryMutexes = append(boundaryMutexes, p.leftBoundaryMutex)
                }
                if newX > x && p.rightBoundaryMutex != nil {
                    boundaryMutexes = append(boundaryMutexes, p.rightBoundaryMutex)
                }
            }
    
            if (y == p.startY && newY < y) || (y == p.endY && newY > y) {
                // Crossing horizontal boundary.
                if newY < y && p.topBoundaryMutex != nil {
                    boundaryMutexes = append(boundaryMutexes, p.topBoundaryMutex)
                }
                if newY > y && p.bottomBoundaryMutex != nil {
                    boundaryMutexes = append(boundaryMutexes, p.bottomBoundaryMutex)
                }
            }
    
            // Sort and lock boundary mutexes to ensure consistent locking order.
            sort.Slice(boundaryMutexes, func(i, j int) bool {
                return uintptr(unsafe.Pointer(boundaryMutexes[i])) < uintptr(unsafe.Pointer(boundaryMutexes[j]))
            })
            for _, mu := range boundaryMutexes {
                mu.Lock()
            }
    
            // Check if the new cell is occupied by a fish.
            if g.grid[newX][newY] != nil && g.grid[newX][newY].GetType() == "fish" {
                // Move the shark to the new position.
                g.grid[x][y] = nil            // Clear the current cell.
                shark.SetPosition(newX, newY) // Update shark's position.
                g.grid[newX][newY] = shark    // Place shark in the new cell.
    
                shark.starve = 0 // Reset the shark's starvation counter.
    
                // Increment the shark's breed timer.
                shark.breedTimer++
                if shark.breedTimer == 5 {
                    // Shark is ready to breed.
                    shark.breedTimer = 0
                    // Create a new shark at the old position.
                    newShark := &Shark{x: x, y: y, breedTimer: 0, starve: 0}
                    g.grid[x][y] = newShark                      // Place the new shark in the old cell.
                    localSharkAdditions = append(localSharkAdditions, newShark) // Add to local additions.
                }
    
                // Mark the fish for removal from the fish slice.
                var fishToRemove *Fish
                for _, fish := range fishCopy {
                    fx, fy := fish.GetPosition()
                    if fx == newX && fy == newY {
                        fishToRemove = fish
                        break
                    }
                }
                if fishToRemove != nil {
                    localFishRemovals = append(localFishRemovals, fishToRemove)
                }
    
                moved = true // Mark that the shark has moved.
            }
    
            // Unlock boundary mutexes in reverse order.
            for i := len(boundaryMutexes) - 1; i >= 0; i-- {
                boundaryMutexes[i].Unlock()
            }
    
            if moved {
                break // Exit the direction loop if the shark has moved.
            }
        }

            if !moved { // Check if the shark hasn't moved yet.
            for dir := 0; dir < 4; dir++ {
                direction := rand.Intn(4) // Randomly select a direction (0-3).
        
                newX, newY := x, y // Initialize the new position variables.
        
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
        
                // Determine if crossing boundaries and identify relevant mutexes.
                var boundaryMutexes []*sync.Mutex
        
                if (x == p.startX && newX < x) || (x == p.endX && newX > x) {
                    // Crossing vertical boundary.
                    if newX < x && p.leftBoundaryMutex != nil {
                        boundaryMutexes = append(boundaryMutexes, p.leftBoundaryMutex)
                    }
                    if newX > x && p.rightBoundaryMutex != nil {
                        boundaryMutexes = append(boundaryMutexes, p.rightBoundaryMutex)
                    }
                }
        
                if (y == p.startY && newY < y) || (y == p.endY && newY > y) {
                    // Crossing horizontal boundary.
                    if newY < y && p.topBoundaryMutex != nil {
                        boundaryMutexes = append(boundaryMutexes, p.topBoundaryMutex)
                    }
                    if newY > y && p.bottomBoundaryMutex != nil {
                        boundaryMutexes = append(boundaryMutexes, p.bottomBoundaryMutex)
                    }
                }
        
                // Sort and lock boundary mutexes to ensure consistent locking order.
                sort.Slice(boundaryMutexes, func(i, j int) bool {
                    return uintptr(unsafe.Pointer(boundaryMutexes[i])) < uintptr(unsafe.Pointer(boundaryMutexes[j]))
                })
                for _, mu := range boundaryMutexes {
                    mu.Lock()
                }
        
                // Check if the new cell is empty.
                if g.grid[newX][newY] == nil {
                    // Move the shark to the new position.
                    g.grid[x][y] = nil            // Clear the current cell.
                    shark.SetPosition(newX, newY) // Update shark's position.
                    g.grid[newX][newY] = shark    // Place shark in the new cell.
        
                    shark.starve++ // Increment the shark's starvation counter.
                    if shark.starve == 5 { // Check if the shark dies of starvation.
                        g.grid[newX][newY] = nil                      // Remove shark from the grid.
                        localSharkRemovals = append(localSharkRemovals, shark) // Mark for removal.
                    } else {
                        // Increment the shark's breed timer.
                        shark.breedTimer++
                        if shark.breedTimer == 6 { // Check if the shark is ready to breed.
                            shark.breedTimer = 0
                            // Create a new shark at the old position.
                            newShark := &Shark{x: x, y: y, breedTimer: 0, starve: 0}
                            g.grid[x][y] = newShark                      // Place the new shark in the old cell.
                            localSharkAdditions = append(localSharkAdditions, newShark) // Add to local additions.
                        }
                    }
                    moved = true // Mark that the shark has moved.
                }
        
                // Unlock boundary mutexes in reverse order to prevent deadlocks.
                for i := len(boundaryMutexes) - 1; i >= 0; i-- {
                    boundaryMutexes[i].Unlock()
                }
        
                if moved {
                    break // Exit the direction loop if the shark has moved.
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

// NewGame initializes a new game instance with a grid of cells divided into four quadrants for multi-threading.
//
// Input:
//   - None.
//
// Output:
//   - *Game: A pointer to the newly initialized Game instance.
//
// Functionality:
// 1. Creates a game instance and sets the start time.
// 2. Divides the grid into four quadrants (top-left, top-right, bottom-left, bottom-right).
//    - Each quadrant has mutexes for managing boundary synchronization between threads.
// 3. Initializes the grid with random entities (fish, sharks, or empty spaces).
//    - Fish and sharks are placed with specified probabilities.
//    - Populates the fish and shark lists for efficient access.
func NewGame() *Game {
    // Initialize a new Game instance with the current start time.
    game := &Game{
        startTime: time.Now(),
    }

    // Define the size of each quadrant along the x and y axes.
    partitionXSize := xdim / 2 // Half the grid width for x-axis division.
    partitionYSize := ydim / 2 // Half the grid height for y-axis division.

    // Create mutexes for managing boundary synchronization.
    verticalBoundaryMutex := &sync.Mutex{}   // Mutex for vertical boundaries (between left and right quadrants).
    horizontalBoundaryMutex := &sync.Mutex{} // Mutex for horizontal boundaries (between top and bottom quadrants).

    // Define partitions for the four quadrants.
    game.partitions = []Partition{
        // Top-left quadrant.
        {
            startX:              0,                        // Start of the x range.
            endX:                partitionXSize - 1,       // End of the x range.
            startY:              0,                        // Start of the y range.
            endY:                partitionYSize - 1,       // End of the y range.
            leftBoundaryMutex:   nil,                      // No left boundary (outer edge).
            rightBoundaryMutex:  verticalBoundaryMutex,    // Mutex for right boundary.
            topBoundaryMutex:    nil,                      // No top boundary (outer edge).
            bottomBoundaryMutex: horizontalBoundaryMutex,  // Mutex for bottom boundary.
        },
        // Top-right quadrant.
        {
            startX:              partitionXSize,           // Start of the x range.
            endX:                xdim - 1,                 // End of the x range.
            startY:              0,                        // Start of the y range.
            endY:                partitionYSize - 1,       // End of the y range.
            leftBoundaryMutex:   verticalBoundaryMutex,    // Mutex for left boundary.
            rightBoundaryMutex:  nil,                      // No right boundary (outer edge).
            topBoundaryMutex:    nil,                      // No top boundary (outer edge).
            bottomBoundaryMutex: horizontalBoundaryMutex,  // Mutex for bottom boundary.
        },
        // Bottom-left quadrant.
        {
            startX:              0,                        // Start of the x range.
            endX:                partitionXSize - 1,       // End of the x range.
            startY:              partitionYSize,           // Start of the y range.
            endY:                ydim - 1,                 // End of the y range.
            leftBoundaryMutex:   nil,                      // No left boundary (outer edge).
            rightBoundaryMutex:  verticalBoundaryMutex,    // Mutex for right boundary.
            topBoundaryMutex:    horizontalBoundaryMutex,  // Mutex for top boundary.
            bottomBoundaryMutex: nil,                      // No bottom boundary (outer edge).
        },
        // Bottom-right quadrant.
        {
            startX:              partitionXSize,           // Start of the x range.
            endX:                xdim - 1,                 // End of the x range.
            startY:              partitionYSize,           // Start of the y range.
            endY:                ydim - 1,                 // End of the y range.
            leftBoundaryMutex:   verticalBoundaryMutex,    // Mutex for left boundary.
            rightBoundaryMutex:  nil,                      // No right boundary (outer edge).
            topBoundaryMutex:    horizontalBoundaryMutex,  // Mutex for top boundary.
            bottomBoundaryMutex: nil,                      // No bottom boundary (outer edge).
        },
    }

    // Populate the grid with random entities (fish, sharks, or empty spaces).
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