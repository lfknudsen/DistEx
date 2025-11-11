# DistEx

Implementation of distributed mutual exclusion using Ricart \& Agrawala



To run use the command `go run node.go PORT`



PORT should be replaced with the specific port you want to use for that node, either 5000, 5001, or 5002



Each node should be opened in a separate terminal, all three ports must be used.



Once opened the node can write to the critical service, a shared textfile, by standard written inputs into the terminal.



When you want to close the node press ctrl+c, alternatively input `quit` or `exit` in the terminal.







