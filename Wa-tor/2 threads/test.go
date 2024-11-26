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
    xdim        = 50                 // Number of cells in the x direction
    ydim        = 50                // Number of cells in the y direction
    windowXSize = 800                // Width of the window in pixels
    windowYSize = 600                // Height of the window in pixels
    cellXSize   = windowXSize / xdim // Width of each cell in pixels
    cellYSize   = windowYSize / ydim // Height of each cell in pixels
)

// Game struct representing the state of the game
type Game struct {
    grid            [xdim][ydim]Entity
    fish            []*Fish
    shark           []*Shark
    startTime       time.Time
    simComplete     bool
    totalFrames     int
    partitions      []Partition
    boundaryMutexes map[int]*sync.Mutex
    fishMutex       sync.Mutex
    sharkMutex      sync.Mutex
    gridMutex       [xdim][ydim]sync.Mutex
}

type Partition struct {
    startX int
    endX   int
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
        writeSimulationDataToCSV("simulation_results.csv", g, len(g.partitions), avgFPS)
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
    // Local slices for additions and removals
    var localFishAdditions []*Fish
    var localFishRemovals []*Fish
    var localSharkAdditions []*Shark
    var localSharkRemovals []*Shark

    // Create copies of g.fish and g.shark to avoid concurrent read issues
    g.fishMutex.Lock()
    fishCopy := make([]*Fish, len(g.fish))
    copy(fishCopy, g.fish)
    g.fishMutex.Unlock()

    g.sharkMutex.Lock()
    sharkCopy := make([]*Shark, len(g.shark))
    copy(sharkCopy, g.shark)
    g.sharkMutex.Unlock()

    // Process fish
    for _, fish := range fishCopy {
        x, y := fish.GetPosition()
        if x < p.startX || x > p.endX {
            continue // Skip fish not in this partition
        }

        moved := false
        for dir := 0; dir < 4; dir++ {
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

            // Check if the new position crosses the partition boundary
            isCrossingBoundary := (newX < p.startX) || (newX > p.endX)
            var mu *sync.Mutex
            if isCrossingBoundary {
                mu = g.boundaryMutexes[newX]
                mu.Lock()
            }

            // Lock the grid cells before modifying
            g.gridMutex[x][y].Lock()
            g.gridMutex[newX][newY].Lock()

            // Check if the new position is empty
            if g.grid[newX][newY] == nil {
                // Move the fish
                g.grid[x][y] = nil
                fish.SetPosition(newX, newY)
                g.grid[newX][newY] = fish
                fish.breedTimer++
                if fish.breedTimer == 5 {
                    fish.breedTimer = 0
                    newFish := &Fish{x: x, y: y, breedTimer: 0}
                    g.grid[x][y] = newFish
                    localFishAdditions = append(localFishAdditions, newFish)
                }
                moved = true
            }

            g.gridMutex[newX][newY].Unlock()
            g.gridMutex[x][y].Unlock()

            if isCrossingBoundary && mu != nil {
                mu.Unlock()
            }

            if moved {
                break
            }
        }
    }

    // Process sharks
    for _, shark := range sharkCopy {
        x, y := shark.GetPosition()
        if x < p.startX || x > p.endX {
            continue // Skip sharks not in this partition
        }

        moved := false

        // Try to move to a position occupied by a fish first
        for dir := 0; dir < 4; dir++ {
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

            // Check if the new position crosses the partition boundary
            isCrossingBoundary := (newX < p.startX) || (newX > p.endX)
            var mu *sync.Mutex
            if isCrossingBoundary {
                mu = g.boundaryMutexes[newX]
                mu.Lock()
            }

            // Lock the grid cells before modifying
            g.gridMutex[x][y].Lock()
            g.gridMutex[newX][newY].Lock()

            // Check if the new position is occupied by a fish
            if g.grid[newX][newY] != nil && g.grid[newX][newY].GetType() == "fish" {
                // Move the shark
                g.grid[x][y] = nil
                shark.SetPosition(newX, newY)
                g.grid[newX][newY] = shark
                shark.starve = 0
                shark.breedTimer++
                if shark.breedTimer == 5 {
                    shark.breedTimer = 0
                    newShark := &Shark{x: x, y: y, breedTimer: 0, starve: 0}
                    g.grid[x][y] = newShark
                    localSharkAdditions = append(localSharkAdditions, newShark)
                }
                // Mark the fish for removal
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
                moved = true
            }

            g.gridMutex[newX][newY].Unlock()
            g.gridMutex[x][y].Unlock()

            if isCrossingBoundary && mu != nil {
                mu.Unlock()
            }

            if moved {
                break
            }
        }

        // If shark didn't move to eat a fish, try to move to an empty cell
        if !moved {
            for dir := 0; dir < 4; dir++ {
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

                // Check if the new position crosses the partition boundary
                isCrossingBoundary := (newX < p.startX) || (newX > p.endX)
                var mu *sync.Mutex
                if isCrossingBoundary {
                    mu = g.boundaryMutexes[newX]
                    mu.Lock()
                }

                // Lock the grid cells before modifying
                g.gridMutex[x][y].Lock()
                g.gridMutex[newX][newY].Lock()

                // Check if the new position is empty
                if g.grid[newX][newY] == nil {
                    // Move the shark
                    g.grid[x][y] = nil
                    shark.SetPosition(newX, newY)
                    g.grid[newX][newY] = shark
                    shark.starve++
                    if shark.starve == 5 {
                        // Shark dies
                        g.grid[newX][newY] = nil
                        localSharkRemovals = append(localSharkRemovals, shark)
                    } else {
                        shark.breedTimer++
                        if shark.breedTimer == 6 {
                            shark.breedTimer = 0
                            newShark := &Shark{x: x, y: y, breedTimer: 0, starve: 0}
                            g.grid[x][y] = newShark
                            localSharkAdditions = append(localSharkAdditions, newShark)
                        }
                    }
                    moved = true
                }

                g.gridMutex[newX][newY].Unlock()
                g.gridMutex[x][y].Unlock()

                if isCrossingBoundary && mu != nil {
                    mu.Unlock()
                }

                if moved {
                    break
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
    game.partitions = []Partition{
        {startX: 0, endX: partitionSize - 1},
        {startX: partitionSize, endX: xdim - 1},
    }

    game.boundaryMutexes = make(map[int]*sync.Mutex)
    for _, p := range game.partitions {
        // Initialize mutexes for both boundaries
        if p.endX+1 < xdim {
            game.boundaryMutexes[p.endX+1] = &sync.Mutex{}
        } else {
            game.boundaryMutexes[0] = &sync.Mutex{} // Wrap-around
        }
        if p.startX-1 >= 0 {
            game.boundaryMutexes[p.startX-1] = &sync.Mutex{}
        } else {
            game.boundaryMutexes[xdim-1] = &sync.Mutex{} // Wrap-around
        }
    }

    // Initialize grid mutexes
    for i := 0; i < xdim; i++ {
        for j := 0; j < ydim; j++ {
            game.gridMutex[i][j] = sync.Mutex{}
        }
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