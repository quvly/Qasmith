OPENQASM 3.0;
include "stdgates.inc";

qubit[4] q;
bit[4] c;

h q[0];
h q[1];
h q[2];
h q[3];
rz(0.1) q[0];
rz(0.2) q[1];
rz(0.3) q[2];
rz(0.4) q[3];

c[0] = measure q[0];
c[1] = measure q[1];
c[2] = measure q[2];
c[3] = measure q[3];
