#!/bin/bash
# Complete TSP CSV workflow demonstration

set -e

# Use the qasmith binary from the repository
QASMITH="/Users/lubosberan/Dev/Qasmith/qasmith"

echo "=========================================="
echo "Qasmith TSP CSV Workflow Demo"
echo "=========================================="
echo ""

# Check if qasmith is available
if [ ! -x "$QASMITH" ]; then
    echo "Error: qasmith binary not found at $QASMITH"
    echo "Please build it first: cd /Users/lubosberan/Dev/Qasmith && go build -o qasmith ./cmd/qasmith"
    exit 1
fi

echo "Step 1: Create a project"
echo "----------------------------------------"
$QASMITH init tsp-demo
cd tsp-demo
echo "✓ Created project: tsp-demo/"
echo ""

echo "Step 2: Create distance matrix CSV"
echo "----------------------------------------"
cat > distances.csv << 'EOF'
# Distance matrix for 4 cities
# Cities: A, B, C, D
# Units: arbitrary distance units
0.00,10.00,15.00,20.00
10.00,0.00,35.00,25.00
15.00,35.00,0.00,30.00
20.00,25.00,30.00,0.00
EOF

echo "✓ Created distances.csv"
cat distances.csv
echo ""

echo "Step 3: Generate Smith code from CSV"
echo "----------------------------------------"
$QASMITH csv-to-smith distances.csv
echo "✓ Generated distances.smith"
echo ""

echo "Step 4: View generated Smith code"
echo "----------------------------------------"
cat distances.smith
echo ""

echo "Step 5: Run the TSP solver"
echo "----------------------------------------"
$QASMITH run distances.smith 2>&1 | grep -A 5 "Best observed"
echo ""

echo "Step 6: Check dependencies"
echo "----------------------------------------"
$QASMITH deps distances.smith
echo ""

echo "=========================================="
echo "Demo Complete!"
echo "=========================================="
echo ""
echo "The workflow successfully:"
echo "  1. Created a Qasmith project"
echo "  2. Read distance data from CSV"
echo "  3. Generated Smith code automatically"
echo "  4. Solved TSP using QAOA"
echo ""
echo "Files created:"
echo "  - distances.csv (input data)"
echo "  - distances.smith (generated code)"
echo ""
echo "Next steps:"
echo "  - Try larger CSV files (up to 5 cities for 16-qubit limit)"
echo "  - Customize QAOA parameters in the generated .smith file"
echo "  - Explore the examples/tsp-csv/ directory for more examples"
echo ""
