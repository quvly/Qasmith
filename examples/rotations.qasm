OPENQASM 3.0;
include "stdgates.inc";

qubit[2] q;
bit[2] c;

rx(3.141592653589793) q[0];
ry(3.141592653589793) q[1];
cx q[0], q[1];
rz(1.5707963267948966) q[1];
cx q[0], q[1];

c[0] = measure q[0];
c[1] = measure q[1];
