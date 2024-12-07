package main

import (
    "encoding/csv"
    "image/color"
    "log"
    "math/rand"
    "os"
    "strconv"
    "sync"
    "time"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// Constants for grid and window dimensions
const (
    xdim        = 400                 // Number of cells in the x direction
    ydim        = 400                 // Number of cells in the y direction
    windowXSize = 800                // Width of the window in pixels
    windowYSize = 800                // Height of the window in pixels
    cellXSize   = windowXSize / xdim // Width of each cell in pixels
    cellYSize   = windowYSize / ydim // Height of each cell in pixels
)

// Game struct representing the state of the game
type Game struct {
    grid        [xdim][ydim]Entity
    fish        []*Fish
    shark       []*Shark
    startTime   time.Time
    simComplete bool
    totalFrames int
    partitions  []Partition
    fishMutex   sync.Mutex
    sharkMutex  sync.Mutex
}

// Partition struct representing a section of the grid
type Partition struct {
    startX             int
    endX               int
    leftBoundaryMutex  *sync.Mutex
    rightBoundaryMutex *sync.Mutex
}

// Entity interface for all game entities (Fish and Shark)
type Entity interface {
    GetType() string
    GetPosition() (int, int)
    SetPosition(x, y int)
}

// Shark struct representing a shark entity
type Shark struct {
    x, y       int
    starve     int
    breedTimer int
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
type Fish struct {
    x, y       int
    breedTimer int
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

// CalculateAverageFPS calculates the average frames per second.
func (g *Game) CalculateAverageFPS() float64 {
    elapsedTime := time.Since(g.startTime).Seconds()
    if elapsedTime > 0 {
        return float64(g.totalFrames) / elapsedTime
    }
    return 0.0
}

// Update function, called every frame to update the game state
func (g *Game) Update() error {
    g.RecordFrame()

    if time.Since(g.startTime) > 10*time.Second {
        g.simComplete = true
        avgFPS := g.CalculateAverageFPS()
        writeSimulationDataToCSV("simulation_results_2_threads.csv", g, len(g.partitions), avgFPS)
        return nil
    }

    var wg sync.WaitGroup
    wg.Add(len(g.partitions))

    // Prepare slices to collect results
    allFishAdditions := make([][]*Fish, len(g.partitions))
    allFishRemovals := make([][]*Fish, len(g.partitions))
    allSharkAdditions := make([][]*Shark, len(g.partitions))
    allSharkRemovals := make([][]*Shark, len(g.partitions))

    for i, partition := range g.partitions {
        go func(i int, p Partition) {
            defer wg.Done()
            fa, fr, sa, sr := g.RunPartition(p)
            allFishAdditions[i] = fa
            allFishRemovals[i] = fr
            allSharkAdditions[i] = sa
            allSharkRemovals[i] = sr
        }(i, partition)
    }

    wg.Wait() // Wait for all partitions to finish

    // Process additions and removals
    g.processRemovalsAndAdditions(allFishAdditions, allFishRemovals, allSharkAdditions, allSharkRemovals)

    return nil
}

func (g *Game) processRemovalsAndAdditions(
    allFishAdditions [][]*Fish, allFishRemovals [][]*Fish,
    allSharkAdditions [][]*Shark, allSharkRemovals [][]*Shark) {

    // Combine slices
    var fishAdditions []*Fish
    var fishRemovals []*Fish
    var sharkAdditions []*Shark
    var sharkRemovals []*Shark

    for _, fa := range allFishAdditions {
        fishAdditions = append(fishAdditions, fa...)
    }
    for _, fr := range allFishRemovals {
        fishRemovals = append(fishRemovals, fr...)
    }
    for _, sa := range allSharkAdditions {
        sharkAdditions = append(sharkAdditions, sa...)
    }
    for _, sr := range allSharkRemovals {
        sharkRemovals = append(sharkRemovals, sr...)
    }

    // Remove fish
    fishToRemove := make(map[*Fish]bool)
    for _, fish := range fishRemovals {
        fishToRemove[fish] = true
    }

    g.fishMutex.Lock()
    var newFish []*Fish
    for _, fish := range g.fish {
        if !fishToRemove[fish] {
            newFish = append(newFish, fish)
        }
    }
    g.fish = newFish

    // Add new fish
    g.fish = append(g.fish, fishAdditions...)
    g.fishMutex.Unlock()

    // Remove sharks
    sharkToRemove := make(map[*Shark]bool)
    for _, shark := range sharkRemovals {
        sharkToRemove[shark] = true
    }

    g.sharkMutex.Lock()
    var newSharks []*Shark
    for _, shark := range g.shark {
        if !sharkToRemove[shark] {
            newSharks = append(newSharks, shark)
        }
    }
    g.shark = newSharks

    // Add new sharks
    g.shark = append(g.shark, sharkAdditions...)
    g.sharkMutex.Unlock()
}

func (g *Game) RunPartition(p Partition) ([]*Fish, []*Fish, []*Shark, []*Shark) {
    // Local slices for additions and removals of fish and sharks
    var localFishAdditions []*Fish
    var localFishRemovals []*Fish
    var localSharkAdditions []*Shark
    var localSharkRemovals []*Shark

    // Create a copy of g.fish to avoid concurrent read issues
    g.fishMutex.Lock()                       // Lock the fish mutex
    fishCopy := make([]*Fish, len(g.fish))   // Create a slice to hold the copy
    copy(fishCopy, g.fish)                   // Copy the fish slice
    g.fishMutex.Unlock()                     // Unlock the fish mutex

    // Create a copy of g.shark to avoid concurrent read issues
    g.sharkMutex.Lock()                      // Lock the shark mutex
    sharkCopy := make([]*Shark, len(g.shark)) // Create a slice to hold the copy
    copy(sharkCopy, g.shark)                 // Copy the shark slice
    g.sharkMutex.Unlock()                    // Unlock the shark mutex

    // Process each fish in the copied fish slice
    for _, fish := range fishCopy {
        x, y := fish.GetPosition()           // Get the current position of the fish

        // Check if the fish is within this partition
        if x < p.startX || x > p.endX {
            continue                         // Skip fish not in this partition
        }

        moved := false                       // Flag to indicate if the fish has moved

        // Try moving the fish in up to four directions
        for dir := 0; dir < 4; dir++ {
            direction := rand.Intn(4)        // Randomly select a direction (0-3)

            newX, newY := x, y               // Initialize new position variables

            // Determine the new position based on the direction
            switch direction {
            case 0: // North
                if y > 0 {
                    newY = y - 1
                } else {
                    newY = ydim - 1          // Wrap around to the bottom
                }
            case 1: // South
                if y < ydim-1 {
                    newY = y + 1
                } else {
                    newY = 0                 // Wrap around to the top
                }
            case 2: // East
                if x < xdim-1 {
                    newX = x + 1
                } else {
                    newX = 0                 // Wrap around to the left
                }
            case 3: // West
                if x > 0 {
                    newX = x - 1
                } else {
                    newX = xdim - 1          // Wrap around to the right
                }
            }

            // Variable to hold the boundary mutex if crossing a boundary
            var mu *sync.Mutex

            // Check if the new position crosses a partition boundary
            if newX < p.startX {
                // Crosses the left boundary; use the left boundary mutex
                mu = p.leftBoundaryMutex
                mu.Lock()                    // Lock the left boundary mutex
            } else if newX > p.endX {
                // Crosses the right boundary; use the right boundary mutex
                mu = p.rightBoundaryMutex
                mu.Lock()                    // Lock the right boundary mutex
            }

            // Check if the new cell is empty
            if g.grid[newX][newY] == nil {
                // Move the fish to the new position
                g.grid[x][y] = nil           // Clear the current cell
                fish.SetPosition(newX, newY) // Update fish's position
                g.grid[newX][newY] = fish    // Place fish in the new cell

                // Increment the fish's breed timer
                fish.breedTimer++
                if fish.breedTimer == 5 {
                    // Fish is ready to breed
                    fish.breedTimer = 0
                    // Create a new fish at the old position
                    newFish := &Fish{x: x, y: y, breedTimer: 0}
                    g.grid[x][y] = newFish               // Place new fish in the old cell
                    localFishAdditions = append(localFishAdditions, newFish) // Add to local additions
                }
                moved = true                // Mark that the fish has moved
            }


            // If a boundary mutex was used, unlock it
            if mu != nil {
                mu.Unlock()
            }

            if moved {
                break                        // Exit the direction loop if the fish has moved
            }
        }
    }

    // Process each shark in the copied shark slice
    for _, shark := range sharkCopy {
        x, y := shark.GetPosition()          // Get the current position of the shark

        // Check if the shark is within this partition
        if x < p.startX || x > p.endX {
            continue                         // Skip sharks not in this partition
        }

        moved := false                       // Flag to indicate if the shark has moved

        // Try to move to a position occupied by a fish first
        for dir := 0; dir < 4; dir++ {
            direction := rand.Intn(4)        // Randomly select a direction (0-3)

            newX, newY := x, y               // Initialize new position variables

            // Determine the new position based on the direction
            switch direction {
            case 0: // North
                if y > 0 {
                    newY = y - 1
                } else {
                    newY = ydim - 1          // Wrap around to the bottom
                }
            case 1: // South
                if y < ydim-1 {
                    newY = y + 1
                } else {
                    newY = 0                 // Wrap around to the top
                }
            case 2: // East
                if x < xdim-1 {
                    newX = x + 1
                } else {
                    newX = 0                 // Wrap around to the left
                }
            case 3: // West
                if x > 0 {
                    newX = x - 1
                } else {
                    newX = xdim - 1          // Wrap around to the right
                }
            }

            // Variable to hold the boundary mutex if crossing a boundary
            var mu *sync.Mutex

            // Check if the new position crosses a partition boundary
            if newX < p.startX {
                // Crosses the left boundary; use the left boundary mutex
                mu = p.leftBoundaryMutex
                mu.Lock()                    // Lock the left boundary mutex
            } else if newX > p.endX {
                // Crosses the right boundary; use the right boundary mutex
                mu = p.rightBoundaryMutex
                mu.Lock()                    // Lock the right boundary mutex
            }

            // Check if the new cell is occupied by a fish
            if g.grid[newX][newY] != nil && g.grid[newX][newY].GetType() == "fish" {
                // Move the shark to the new position
                g.grid[x][y] = nil           // Clear the current cell
                shark.SetPosition(newX, newY)// Update shark's position
                g.grid[newX][newY] = shark   // Place shark in the new cell

                shark.starve = 0             // Reset the shark's starvation counter

                // Increment the shark's breed timer
                shark.breedTimer++
                if shark.breedTimer == 5 {
                    // Shark is ready to breed
                    shark.breedTimer = 0
                    // Create a new shark at the old position
                    newShark := &Shark{x: x, y: y, breedTimer: 0, starve: 0}
                    g.grid[x][y] = newShark              // Place new shark in the old cell
                    localSharkAdditions = append(localSharkAdditions, newShark) // Add to local additions
                }

                // Mark the fish for removal from the fish slice
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

                moved = true                // Mark that the shark has moved
            }


            // If a boundary mutex was used, unlock it
            if mu != nil {
                mu.Unlock()
            }

            if moved {
                break                        // Exit the direction loop if the shark has moved
            }
        }

        // If the shark didn't move by eating a fish, try to move to an empty cell
        if !moved {
            for dir := 0; dir < 4; dir++ {
                direction := rand.Intn(4)    // Randomly select a direction (0-3)

                newX, newY := x, y           // Initialize new position variables

                // Determine the new position based on the direction
                switch direction {
                case 0: // North
                    if y > 0 {
                        newY = y - 1
                    } else {
                        newY = ydim - 1      // Wrap around to the bottom
                    }
                case 1: // South
                    if y < ydim-1 {
                        newY = y + 1
                    } else {
                        newY = 0             // Wrap around to the top
                    }
                case 2: // East
                    if x < xdim-1 {
                        newX = x + 1
                    } else {
                        newX = 0             // Wrap around to the left
                    }
                case 3: // West
                    if x > 0 {
                        newX = x - 1
                    } else {
                        newX = xdim - 1      // Wrap around to the right
                    }
                }

                // Variable to hold the boundary mutex if crossing a boundary
                var mu *sync.Mutex

                // Check if the new position crosses a partition boundary
                if newX < p.startX {
                    // Crosses the left boundary; use the left boundary mutex
                    mu = p.leftBoundaryMutex
                    mu.Lock()                // Lock the left boundary mutex
                } else if newX > p.endX {
                    // Crosses the right boundary; use the right boundary mutex
                    mu = p.rightBoundaryMutex
                    mu.Lock()                // Lock the right boundary mutex
                }

                // Check if the new cell is empty
                if g.grid[newX][newY] == nil {
                    // Move the shark to the new position
                    g.grid[x][y] = nil       // Clear the current cell
                    shark.SetPosition(newX, newY)// Update shark's position
                    g.grid[newX][newY] = shark   // Place shark in the new cell

                    shark.starve++           // Increment the shark's starvation counter
                    if shark.starve == 5 {
                        // Shark dies of starvation
                        g.grid[newX][newY] = nil // Remove shark from the grid
                        localSharkRemovals = append(localSharkRemovals, shark) // Mark for removal
                    } else {
                        // Increment the shark's breed timer
                        shark.breedTimer++
                        if shark.breedTimer == 6 {
                            // Shark is ready to breed
                            shark.breedTimer = 0
                            // Create a new shark at the old position
                            newShark := &Shark{x: x, y: y, breedTimer: 0, starve: 0}
                            g.grid[x][y] = newShark          // Place new shark in the old cell
                            localSharkAdditions = append(localSharkAdditions, newShark) // Add to local additions
                        }
                    }
                    moved = true            // Mark that the shark has moved
                }


                // If a boundary mutex was used, unlock it
                if mu != nil {
                    mu.Unlock()
                }

                if moved {
                    break                    // Exit the direction loop if the shark has moved
                }
            }
        }
    }

    // Return local additions and removals
    return localFishAdditions, localFishRemovals, localSharkAdditions, localSharkRemovals
}

// Draw function, called every frame to render the game screen
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
                    rectColor = color.RGBA{0, 221, 255, 255} // Blue for fish
                case "shark":
                    rectColor = color.RGBA{190, 44, 190, 255} // Purple for shark
                }
            } else {
                rectColor = color.RGBA{0, 0, 0, 255} // Black for empty
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
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
    return windowXSize, windowYSize
}

// NewGame function initializes a new game instance with a grid of cells
func NewGame() *Game {
    game := &Game{
        startTime: time.Now(),
    }

    partitionSize := xdim / 2 // For two threads

    // Create boundary mutexes
    leftBoundaryMutex := &sync.Mutex{}
    rightBoundaryMutex := &sync.Mutex{}

    game.partitions = []Partition{
        {
            startX:             0,
            endX:               partitionSize - 1,
            leftBoundaryMutex:  leftBoundaryMutex,
            rightBoundaryMutex: rightBoundaryMutex,
        },
        {
            startX:             partitionSize,
            endX:               xdim - 1,
            leftBoundaryMutex:  rightBoundaryMutex,
            rightBoundaryMutex: leftBoundaryMutex,
        },
    }


    // Initialize grid with random fish or empty spaces
    for i := 0; i < xdim; i++ {
        for k := 0; k < ydim; k++ {
            randomNum := rand.Intn(100) + 1 // Random number between 1 and 100
            if randomNum >= 5 && randomNum <= 10 {
                // Create and place a fish
                fish := &Fish{x: i, y: k, breedTimer: 0}
                game.grid[i][k] = fish
                game.fish = append(game.fish, fish)
            } else if randomNum == 86 {
                // Create and place a shark
                shark := &Shark{x: i, y: k, breedTimer: 0, starve: 0}
                game.grid[i][k] = shark
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

func writeSimulationDataToCSV(filename string, g *Game, partitions int, frameRate float64) {
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
        strconv.Itoa(xdim * ydim),
        strconv.Itoa(len(g.partitions)), // Convert the thread count to a string
        strconv.FormatFloat(frameRate, 'f', 2, 64), // Convert the frame rate to a string with 2 decimal places
    }
    // Write the prepared data to the CSV file
    if err := writer.Write(data); err != nil {
        // Log an error if the data cannot be written to the file
        log.Fatalf("failed to write to csv: %v", err)
    }
}