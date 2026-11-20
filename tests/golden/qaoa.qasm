OPENQASM 3.0;
include "stdgates.inc";

qubit[2] q;
bit[2] c;

h q[0];
h q[1];
rz(-2.0) q[1];
cx q[0], q[1];
rz(0.8) q[1];
cx q[0], q[1];
rx(0.6) q[0];
rx(0.6) q[1];
rz(-1.0) q[1];
cx q[0], q[1];
rz(0.4) q[1];
cx q[0], q[1];
rx(1.2) q[0];
rx(1.2) q[1];

c[0] = measure q[0];
c[1] = measure q[1];
