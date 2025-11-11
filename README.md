# DistEx

Implementation of distributed mutual exclusion using the Ricart \& Agrawala
algorithm.

To run, use the command

```
go run node.go <PORT>
```

`<PORT>` should be replaced with the specific port you want to use for that node,
either **5000**, **5001**, or **5002**.

You can also compile it to an executable with
```
go build
```
and then run that with
```
./DistEx <PORT>
```

Each node should be opened in a separate terminal window. They do not all need
to be open at once for the programme to function.

Once opened the node can write to the critical service (a shared plaintext file)
by standard written inputs into the terminal.

To close the current node, input `quit` or `exit` into the terminal, or kill the
process with Ctrl+C.







