# Real-World TSP Solver with CSV Data

This example demonstrates how to solve the Traveling Salesman Problem (TSP) using real-world distance data loaded from CSV files.

## Overview

The workflow consists of three steps:

1. **Prepare your data** - Create a CSV file with distance matrix
2. **Generate Smith code** - Use `qasmith csv-to-smith` to convert CSV to Smith
3. **Run the solver** - Execute the generated Smith program with QAOA

## Step 1: Prepare Your Data

Create a CSV file with a symmetric distance matrix. The matrix should be square (N×N for N cities).

**Example CSV format:**

```csv
# Distance matrix for 5 cities
# Distances are Euclidean distances between city coordinates
0.00,355.47,850.25,542.34,803.00
355.47,0.00,498.83,258.90,447.87
850.25,498.83,0.00,468.71,131.06
542.34,258.90,468.71,0.00,359.93
803.00,447.87,131.06,359.93,0.00
```

**Notes:**
- Lines starting with `#` are comments and will be ignored
- All values must be numeric (integers or floats)
- The matrix must be square
- Diagonal elements (distance from city to itself) should be 0

## Step 2: Generate Smith Code

Use the `csv-to-smith` command to convert your CSV to a Smith program:

```bash
qasmith csv-to-smith distances.csv
```

This generates a `.smith` file with the same base name:

```bash
# Input:  distances.csv
# Output: distances.smith
```

You can also specify an output file:

```bash
qasmith csv-to-smith distances.csv -o my_tsp.smith
```

## Step 3: Run the Solver

Execute the generated Smith program:

```bash
qasmith run distances.smith
```

The program uses QAOA (Quantum Approximate Optimization Algorithm) to find an approximate solution to the TSP.

## Example: 5-City TSP

This repository includes a complete 5-city example:

```bash
cd examples/tsp-csv

# View the distance matrix
cat distances_5.csv

# Generate Smith code
qasmith csv-to-smith distances_5.csv

# Run the solver
qasmith run distances_5.smith
```

**Expected output:**
```
Qubits: 16
Layers: 1
Shots: 1000
Estimated expected QUBO energy: 141893.15973
Measurement counts:
...
Best observed assignment: 0010100010000100
Best observed QUBO energy: 19155.32
Classical optimal route: [0 1 2 4 3 0]
Classical optimal distance: 1887.63
```

## Simulator Limitations

The current Qasmith simulator has a **16-qubit limit**. The TSP encoding uses `(N-1)²` qubits for N cities:

| Cities | Qubits Required | Status |
|--------|----------------|--------|
| 3 | 4 | ✅ Within limit |
| 4 | 9 | ✅ Within limit |
| 5 | 16 | ✅ At limit |
| 6 | 25 | ❌ Exceeds limit |
| 10 | 81 | ❌ Exceeds limit |
| 50 | 2401 | ❌ Exceeds limit |

**For larger problems (>5 cities):**
- The Smith code can still be generated
- Compilation will succeed
- But simulation will fail due to qubit limits
- For production use, you would need a larger quantum simulator or quantum hardware

## Generating Test Data

The repository includes Python scripts to generate test data:

```bash
cd examples/tsp-csv

# Generate 5-city example (within simulator limits)
python3 generate_tsp_data_5.py

# Generate larger examples (for demonstration)
python3 generate_tsp_data.py
```

These scripts create:
- `cities.csv` - City coordinates (id, name, x, y)
- `distances_N.csv` - Distance matrices for various sizes

## Understanding the Output

The QAOA solver produces:

1. **Measurement counts** - Distribution of quantum measurement outcomes
2. **Best observed assignment** - The best bitstring found
3. **QUBO energy** - The cost function value (lower is better)
4. **Feasible samples** - Solutions that satisfy TSP constraints
5. **Classical optimal** - The true optimal solution (computed classically for comparison)

**Note:** QAOA is a heuristic algorithm. With random parameters, it may not always find the optimal solution. Parameter optimization (tuning gamma and beta values) improves results.

## Customizing QAOA Parameters

The generated Smith code uses default QAOA parameters:

```smith
return qaoa(problem, [0.4], [0.3]);
```

You can modify these parameters in the generated `.smith` file:

```smith
// Single layer with custom parameters
return qaoa(problem, [0.5], [0.2]);

// Two layers
return qaoa(problem, [0.5, 0.3], [0.2, 0.4]);
```

## Complete Workflow Example

```bash
# 1. Create a project
qasmith init my-tsp-solver
cd my-tsp-solver

# 2. Create your distance matrix CSV
cat > distances.csv << 'EOF'
# 4-city distance matrix
0.0,10.0,15.0,20.0
10.0,0.0,35.0,25.0
15.0,35.0,0.0,30.0
20.0,25.0,30.0,0.0
EOF

# 3. Generate Smith code
qasmith csv-to-smith distances.csv

# 4. Run the solver
qasmith run distances.smith

# 5. View the generated code
cat distances.smith
```

## CSV Format Specification

**Required:**
- Square matrix (N rows, N columns)
- Numeric values (integers or floats)
- Comma-separated values

**Optional:**
- Comment lines starting with `#`
- Empty lines (ignored)

**Example with comments:**
```csv
# Distance matrix for delivery locations
# Units: kilometers
# Source: Google Maps API

0.0,5.2,8.7,12.1
5.2,0.0,6.3,9.8
8.7,6.3,0.0,4.5
12.1,9.8,4.5,0.0
```

## Troubleshooting

**Error: "CSV is not a square matrix"**
- Ensure your CSV has the same number of rows and columns
- Check for missing or extra values

**Error: "distance matrix must have at least 3 cities"**
- TSP requires at least 3 cities
- Add more rows/columns to your CSV

**Error: "simulator limit is 16 qubits"**
- Your problem requires more than 16 qubits
- For N cities, you need (N-1)² qubits
- Reduce the number of cities or use a larger simulator

**No feasible solutions found**
- QAOA parameters need optimization
- Try different gamma/beta values
- Increase the number of QAOA layers

## Next Steps

- Learn about [QAOA optimization](../optimization/qaoa.md)
- Explore [TSP encoding details](../optimization/tsp.md)
- Try [custom QUBO models](../optimization/custom-models.md)
