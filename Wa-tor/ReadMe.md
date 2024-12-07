Wa-Tor is a concurrent simulation of predator-prey dynamics in a toroidal world. This project demonstrates multi-threaded programming concepts and showcases the interactions between sharks and fish in a constrained environment, with real-time visualisation powered by Ebiten.

## License

Wa-Tor © 2024 by Ronan Green is licensed under CC BY-NC 4.0. To view a copy of this license, visit [https://creativecommons.org/licenses/by-nc/4.0/](https://creativecommons.org/licenses/by-nc/4.0/).

## Authors

- Ronan Green
    

## Features

- Multi-threaded simulation with up to 8 threads for partitioned execution.
    
- Real-time visualisation using Ebiten.
    
- Configurable grid size and simulation parameters.
    
- Dynamic shark and fish populations with breeding, movement, and starvation mechanics.
    
- Boundary synchronisation using mutexes to handle partitioned grids.
    

## How to Install

1. Clone the repository:
    
    ```
    git clone https://github.com/RonanGreen1/ConDev/tree/main/Wa-tor
    ```
    
2. Use a Golang IDE to open the project (e.g., GoLand or VS Code).
    
3. Run the main file:
    
    ```
    go run main.go
    ```
    

## Prerequisites

- Go programming language (version 1.18 or later).
    
- Golang IDE (recommended).
    

## Libraries Used

- **Ebiten**: For rendering the simulation grid in real-time. ([GitHub Repository](https://github.com/hajimehoshi/ebiten))
    
- **sync**: For managing concurrency using mutexes.
    
- **unsafe**: For fine-grained control in boundary management.
    

## Challenges Faced

- **Concurrency Management**: Implementing safe access to shared data structures required extensive use of mutexes.
    
- **Boundary Synchronisation**: Synchronising shared boundaries between partitions was complex, particularly with diagonal crossings.
    
- **Performance Optimisation**: Balancing granularity of partitions and thread overhead.
    
- **Visual Debugging**: Ensuring the visual representation aligned with the simulation logic.
    

## Design Highlights

- **Toroidal World**: The simulation wraps around at edges, creating a seamless world.
    
- **Partitioning**: The grid is divided into multiple partitions for parallel processing, with boundary mutexes ensuring thread safety.
    
- **Dynamic Entities**: Sharks and fish have unique behaviours like breeding, movement, and starvation, influencing population dynamics.
    

## Usage

1. Run the simulation:
    
    ```
    go run main.go
    ```
    
2. View the simulation window where sharks, fish, and empty spaces are represented by colours.
    
3. Adjust grid dimensions or simulation parameters in the source code to experiment with different configurations.
    

## Output

- The simulation generates a CSV file named `simulation_results.csv` containing:
    
    - Grid size.
        
    - Thread count.
        
    - Average frame rate (FPS).
        