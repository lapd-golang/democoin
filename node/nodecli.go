package main

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/gelembjuk/democoin/lib"
	"github.com/gelembjuk/democoin/lib/nodeclient"
	"github.com/gelembjuk/democoin/lib/wallet"
)

type NodeCLI struct {
	Input              AppInput
	Logger             *lib.LoggerMan
	DataDir            string
	Command            string
	AlreadyRunningPort int
	NodeAuthStr        string
	Node               *Node
}

/*
* Creates a client object
 */
func getNodeCLI(input AppInput) NodeCLI {
	cli := NodeCLI{}
	cli.Input = input
	cli.DataDir = input.DataDir
	cli.Command = input.Command

	cli.Logger = lib.CreateLogger()

	if input.Args.LogDest != "stdout" {
		cli.Logger.LogToFiles(cli.DataDir, "log_trace.txt", "log_info.txt", "log_warning.txt", "log_error.txt")
	}

	cli.Node = nil
	// check if Daemon is already running
	nd := NodeDaemon{}
	nd.DataDir = cli.DataDir
	nd.Logger = cli.Logger

	_, port, authstr, _, err := nd.loadPIDFile()

	if err == nil && port > 0 {
		cli.AlreadyRunningPort = port
		cli.NodeAuthStr = authstr
	} else {
		cli.AlreadyRunningPort = 0
		cli.NodeAuthStr = ""
	}

	cli.Logger.Trace.Println("Node CLI inited")

	return cli
}

/*
* Createes node object. Node does all work related to acces to bockchain and DB
 */
func (c *NodeCLI) CreateNode() {
	if c.Node != nil {
		//already created
		return
	}
	node := Node{}

	node.DataDir = c.DataDir
	node.Logger = c.Logger
	node.MinterAddress = c.Input.MinterAddress

	node.Init()
	node.InitNodes(c.Input.Nodes, false)

	node.NodeClient.SetAuthStr(c.NodeAuthStr)

	c.Node = &node
}

/*
* Detects if this request is not related to node server management and must return response right now
 */
func (c NodeCLI) isInteractiveMode() bool {
	commands := []string{
		"createblockchain",
		"initblockchain",
		"printchain",
		"makeblock",
		"reindexcache",
		"send",
		"getbalance",
		"getbalances",
		"createwallet",
		"listaddresses",
		"unapprovedtransactions",
		"mineblock",
		"canceltransaction",
		"dropblock",
		"addrhistory",
		"showunspent",
		"shownodes",
		"addnode",
		"removenode"}

	for _, cm := range commands {
		if cm == c.Command {
			return true
		}
	}
	return false
}

/*
* Detects if it is a node management command
 */
func (c NodeCLI) isNodeManageMode() bool {

	if "startnode" == c.Command ||
		"startintnode" == c.Command ||
		"stopnode" == c.Command ||
		daemonprocesscommandline == c.Command ||
		"nodestate" == c.Command {
		return true
	}
	return false
}

/*
* Executes the client command in interactive mode
 */
func (c NodeCLI) ExecuteCommand() error {
	c.CreateNode() // init node struct

	if c.Command != "createblockchain" &&
		c.Command != "initblockchain" &&
		c.Command != "createwallet" &&
		c.Command != "listaddresses" {
		// only these 3 addresses can be executed if no blockchain yet
		if !c.Node.BlockchainExist() {
			return errors.New("Blockchain is not found. Must be created or inited")
		}
	}

	if c.Command == "createblockchain" {
		return c.commandCreateBlockchain()

	} else if c.Command == "initblockchain" {
		return c.commandInitBlockchain()

	} else if c.Command == "printchain" {
		return c.commandPrintChain()

	} else if c.Command == "reindexcache" {
		return c.commandReindexCache()

	} else if c.Command == "getbalance" {
		return c.commandGetBalance()

	} else if c.Command == "getbalances" {
		return c.commandAddressesBalance()

	} else if c.Command == "listaddresses" {
		return c.forwardCommandToWallet()

	} else if c.Command == "createwallet" {
		return c.forwardCommandToWallet()

	} else if c.Command == "send" {
		return c.commandSend()

	} else if c.Command == "unapprovedtransactions" {
		return c.commandUnapprovedTransactions()

	} else if c.Command == "makeblock" {
		return c.commandMakeBlock()

	} else if c.Command == "dropblock" {
		return c.commandDropBlock()

	} else if c.Command == "canceltransaction" {
		return c.commandCancelTransaction()

	} else if c.Command == "addrhistory" {
		return c.commandAddressHistory()

	} else if c.Command == "showunspent" {
		return c.commandShowUnspent()

	} else if c.Command == "shownodes" {
		return c.commandShowNodes()

	} else if c.Command == "addnode" {
		return c.commandAddNode()

	} else if c.Command == "removenode" {
		return c.commandRemoveNode()
	}

	return errors.New("Unknown management command")
}

/*
* Creates node server daemon manager
 */
func (c NodeCLI) createDaemonManager() (*NodeDaemon, error) {
	nd := NodeDaemon{}

	c.CreateNode()

	if !c.Node.BlockchainExist() {
		return nil, errors.New("Blockchain is not found. Must be created or inited")
	}

	nd.DataDir = c.DataDir
	nd.Logger = c.Logger
	nd.Port = c.Input.Port
	nd.Host = c.Input.Host
	nd.Node = c.Node
	nd.Init()

	return &nd, nil
}

/*
* Execute server management command
 */
func (c NodeCLI) ExecuteManageCommand() error {
	noddaemon, err := c.createDaemonManager()

	if err != nil {
		return err
	}

	if c.Command == "startnode" {
		return noddaemon.StartServer()

	} else if c.Command == "startintnode" {
		return noddaemon.StartServerInteractive()

	} else if c.Command == "stopnode" {
		return noddaemon.StopServer()

	} else if c.Command == daemonprocesscommandline {
		return noddaemon.DaemonizeServer()

	} else if c.Command == "nodestate" {
		c.CreateNode()
		return c.commandShowState(noddaemon)

	}
	return errors.New("Unknown node manage command")
}

/*
* Creates wallet object for operation related to wallets list management
 */
func (c *NodeCLI) getWalletsCLI() (*wallet.WalletCLI, error) {
	winput := wallet.AppInput{}
	winput.Command = c.Input.Command
	winput.Address = c.Input.Args.Address
	winput.DataDir = c.Input.DataDir
	winput.NodePort = c.Input.Port
	winput.NodeHost = "localhost"
	winput.Amount = c.Input.Args.Amount
	winput.ToAddress = c.Input.Args.To

	if c.Input.Args.From != "" {
		winput.Address = c.Input.Args.From
	}
	c.Logger.Trace.Println("Running port ", c.AlreadyRunningPort)

	walletscli := wallet.WalletCLI{}

	if c.AlreadyRunningPort > 0 {
		winput.NodePort = c.AlreadyRunningPort
		winput.NodeHost = "localhost"
	}

	walletscli.Init(c.Logger, winput)

	walletscli.NodeMode = true

	return &walletscli, nil
}

/*
* Forwards a command to wallet object. This is needed for cases when a node must do some
* operation with local wallets
 */
func (c *NodeCLI) forwardCommandToWallet() error {
	walletscli, err := c.getWalletsCLI()

	if err != nil {
		return err
	}
	c.Logger.Trace.Println("Execute command as a client")
	return walletscli.ExecuteCommand()
}

/*
* Forwards a command to wallet object. This is needed for cases when a node must do some
* operation with local wallets
 */
func (c *NodeCLI) getLocalNetworkClient() nodeclient.NodeClient {
	nc := *c.Node.NodeClient
	nc.NodeAddress.Port = c.AlreadyRunningPort
	nc.NodeAddress.Host = "localhost"
	return nc
}

/*
* To create new blockchain from scratch
 */
func (c *NodeCLI) commandCreateBlockchain() error {
	err := c.Node.CreateBlockchain(c.Input.Args.Address, c.Input.Args.Genesis)

	if err != nil {
		return err
	}

	fmt.Println("Done!")

	return nil
}

/*
* To init blockchain loaded from other node. Is executed for new nodes if blockchain already exists
 */
func (c *NodeCLI) commandInitBlockchain() error {
	// try to open existent BC to check if it exists
	err := c.Node.OpenBlockchain()

	if err == nil {
		c.Node.CloseBlockchain()
		return errors.New("Blockchain already exists")
	}

	alldone, err := c.Node.InitBlockchainFromOther(c.Input.Args.NodeHost, c.Input.Args.NodePort)

	if err != nil {
		return err
	}
	if alldone {
		fmt.Println("Done! ")
	} else {
		fmt.Println("Done! First part of bockchain loaded. Next part will be loaded on background when node started")
	}

	return nil
}

/*
* Print fulll blockchain
 */
func (c *NodeCLI) commandPrintChain() error {
	err := c.Node.OpenBlockchain()

	if err != nil {
		return err
	}
	defer c.Node.CloseBlockchain()

	bci, err := c.Node.NodeBC.GetBlockChainIterator()

	if err != nil {
		return err
	}

	defer bci.Close()

	blocks := []*BlockInfo{}

	for {
		block := bci.Next()

		if c.Input.Args.View == "short" {
			fmt.Printf("===============\n")
			fmt.Printf("Hash: %x\n", block.Hash)
			fmt.Printf("Height: %d, Transactions: %d\n", block.Height, len(block.Transactions)-1)
			fmt.Printf("Prev: %x\n", block.PrevBlockHash)

			fmt.Printf("\n")
		} else if c.Input.Args.View == "shortr" {
			blocks = append(blocks, &block)
		} else {
			fmt.Printf("============ Block %x ============\n", block.Hash)
			fmt.Printf("Height: %d\n", block.Height)
			fmt.Printf("Prev. block: %x\n", block.PrevBlockHash)

			for _, tx := range block.Transactions {
				fmt.Println(tx)
			}
			fmt.Printf("\n\n")
		}
		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	if c.Input.Args.View == "shortr" {
		for i := len(blocks) - 1; i >= 0; i-- {
			block := blocks[i]
			fmt.Printf("===============\n")
			fmt.Printf("Hash: %x\n", block.Hash)
			fmt.Printf("Height: %d, TX count: %d\n", block.Height, len(block.Transactions))
			fmt.Printf("Prev: %x\n", block.PrevBlockHash)

			fmt.Printf("\n")
		}
	}

	return nil
}

/*
* Show contacts of a cache of unapproved transactions
 */
func (c *NodeCLI) commandUnapprovedTransactions() error {
	err := c.Node.OpenBlockchain()

	if err != nil {
		return err
	}
	defer c.Node.CloseBlockchain()

	total, _ := c.Node.NodeTX.IterateUnapprovedTransactions(
		func(txhash, txstr string) {
			fmt.Printf("============ Transaction %x ============\n", txhash)

			fmt.Println(txstr)
		})
	fmt.Printf("\nTotal transactions: %d\n", total)
	return nil
}

/*
* Show all wallets and balances for each of them
 */
func (c *NodeCLI) commandAddressesBalance() error {
	if c.AlreadyRunningPort > 0 {
		// run in wallet mode.
		return c.forwardCommandToWallet()
	}

	err := c.Node.OpenBlockchain()

	if err != nil {
		return err
	}
	defer c.Node.CloseBlockchain()

	walletscli, err := c.getWalletsCLI()

	if err != nil {
		return err
	}
	// get addresses in local wallets

	result, err := c.Node.NodeTX.GetAddressesBalance(walletscli.WalletsObj.GetAddresses())

	if err != nil {
		return err
	}
	fmt.Println("Balance for all addresses:")
	fmt.Println()
	for address, balance := range result {
		fmt.Printf("%s: %.8f (Approved - %.8f, Pending - %.8f)\n", address, balance.Total, balance.Approved, balance.Pending)
	}

	return nil
}

/*
* SHow history for a wallet
 */
func (c *NodeCLI) commandAddressHistory() error {
	if c.AlreadyRunningPort > 0 {
		c.Input.Command = "showhistory"
		// run in wallet mode.
		return c.forwardCommandToWallet()
	}

	err := c.Node.OpenBlockchain()

	if err != nil {
		return err
	}
	defer c.Node.CloseBlockchain()

	result, err := c.Node.NodeBC.GetAddressHistory(c.Input.Args.Address)

	if err != nil {
		return err
	}
	fmt.Println("History of transactions:")
	for _, rec := range result {
		if rec.IOType {
			fmt.Printf("%f\t In from\t%s\n", rec.Value, rec.Address)
		} else {
			fmt.Printf("%f\t Out To  \t%s\n", rec.Value, rec.Address)
		}

	}

	return nil
}

/*
* Show unspent transactions outputs for address
 */
func (c *NodeCLI) commandShowUnspent() error {
	if c.AlreadyRunningPort > 0 {
		// run in wallet mode.
		return c.forwardCommandToWallet()
	}

	err := c.Node.OpenBlockchain()

	if err != nil {
		return err
	}
	defer c.Node.CloseBlockchain()

	result, err := c.Node.NodeTX.UnspentTXs.GetUnspentTransactionsOutputs(c.Input.Args.Address)

	if err != nil {
		return err
	}

	balance := float64(0)

	for _, rec := range result {
		var addr string
		if len(rec.SendPubKeyHash) > 0 {
			addr, _ = lib.PubKeyHashToAddres(rec.SendPubKeyHash)
		} else {
			addr = "Coint base"
		}

		fmt.Printf("%f\t from\t%s in transaction %s output #%d\n", rec.Value, addr, hex.EncodeToString(rec.TXID), rec.OIndex)
		balance += rec.Value
	}

	fmt.Printf("\nBalance - %f\n", balance)

	return nil
}

/*
* Display balance for address
 */
func (c *NodeCLI) commandGetBalance() error {
	if c.AlreadyRunningPort > 0 {
		// run in wallet mode.
		return c.forwardCommandToWallet()
	}

	err := c.Node.OpenBlockchain()

	if err != nil {
		return err
	}
	defer c.Node.CloseBlockchain()

	balance, err := c.Node.NodeTX.GetAddressBalance(c.Input.Args.Address)

	if err != nil {
		return err
	}

	fmt.Printf("Balance of '%s': \nTotal - %.8f\n", c.Input.Args.Address, balance.Total)
	fmt.Printf("Approved - %.8f\n", balance.Approved)
	fmt.Printf("Pending - %.8f\n", balance.Pending)
	return nil
}

/*
* Send money to other address
 */
func (c *NodeCLI) commandSend() error {
	if c.AlreadyRunningPort > 0 {

		// run in wallet mode.
		return c.forwardCommandToWallet()
	}
	c.Logger.Trace.Println("Send with dirct access to DB ")
	err := c.Node.OpenBlockchain()

	if err != nil {
		return err
	}
	defer c.Node.CloseBlockchain()
	// else, access directtly to the DB

	walletscli, err := c.getWalletsCLI()

	if err != nil {
		return err
	}

	walletobj, err := walletscli.WalletsObj.GetWallet(c.Input.Args.From)

	if err != nil {
		return err
	}

	txid, err := c.Node.Send(walletobj.GetPublicKey(), walletobj.GetPrivateKey(),
		c.Input.Args.To, c.Input.Args.Amount)

	if err != nil {
		return err
	}

	fmt.Printf("Success. New transaction: %x\n", txid)

	return nil
}

/*
* Reindex DB of unspent transactions and transaction pointers
 */
func (c *NodeCLI) commandReindexCache() error {
	err := c.Node.OpenBlockchain()

	if err != nil {
		return err
	}
	defer c.Node.CloseBlockchain()

	err = c.Node.NodeTX.TXCache.Reindex()

	if err != nil {
		return err
	}

	count, err := c.Node.NodeTX.UnspentTXs.Reindex()

	if err != nil {
		return err
	}

	fmt.Printf("Done! There are %d transactions in the UTXO set.\n", count)
	return nil
}

/*
* Try to mine a block if there is anough unapproved transactions
 */
func (c *NodeCLI) commandMakeBlock() error {
	block, err := c.Node.TryToMakeBlock()

	if err != nil {
		return err
	}

	if len(block) > 0 {
		fmt.Printf("Done! New block mined with the hash %x.\n", block)
	} else {
		fmt.Printf("Not enough transactions to mine a block.\n")
	}

	return nil
}

/*
* Cancel transaction if it is not yet in a block
 */
func (c *NodeCLI) commandCancelTransaction() error {
	err := c.Node.OpenBlockchain()

	if err != nil {
		return err
	}
	defer c.Node.CloseBlockchain()

	err = c.Node.NodeTX.CancelTransaction(c.Input.Args.Transaction)

	if err != nil {
		return err
	}

	fmt.Printf("Done!\n")
	fmt.Printf("NOTE. This canceled transaction only from local node. If it was already sent to other nodes, than a transaction still can be completed!\n")

	return nil
}

/*
* Drops last block from the top of blockchain
 */
func (c *NodeCLI) commandDropBlock() error {
	err := c.Node.OpenBlockchain()

	if err != nil {
		return err
	}
	defer c.Node.CloseBlockchain()

	err = c.Node.DropBlock()

	if err != nil {
		return err
	}

	bci, err := c.Node.NodeBC.GetBlockChainIterator()

	if err != nil {
		return err
	}

	defer bci.Close()

	block := bci.Next()

	fmt.Printf("Done!\n")
	fmt.Printf("============ Last Block %x ============\n", block.Hash)
	fmt.Printf("Height: %d\n", block.Height)
	fmt.Printf("Prev. block: %x\n", block.PrevBlockHash)

	for _, tx := range block.Transactions {
		fmt.Println(tx)
	}
	fmt.Printf("\n\n")

	return nil
}

/*
* Shows server state
 */
func (c *NodeCLI) commandShowState(daemon *NodeDaemon) error {
	Runnning, ProcessID, Port, err := daemon.GetServerState()

	fmt.Println("Node Server State:")

	if Runnning {
		fmt.Printf("Server is running. Process: %d, listening on the port %d\n", ProcessID, Port)
	} else {
		fmt.Println("Server is not running")
	}

	err = c.Node.OpenBlockchain()

	if err != nil {
		return err
	}
	defer c.Node.CloseBlockchain()

	fmt.Println("Blockchain state:")

	bh, err := c.Node.NodeBC.GetBestHeight()

	if err != nil {
		return err
	}

	fmt.Printf("  Number of blocks - %d\n", bh+1)

	unappr, err := c.Node.NodeTX.UnapprovedTXs.GetCount()

	if err != nil {
		return err
	}

	fmt.Printf("  Number of unapproved transactions - %d\n", unappr)

	return nil
}

// Displays list of nodes (connections)
func (c *NodeCLI) commandShowNodes() error {
	var nodes []lib.NodeAddr
	var err error

	if c.AlreadyRunningPort > 0 {
		// connect to node to get nodes list
		nc := c.getLocalNetworkClient()
		nodes, err = nc.SendGetNodes()

		if err != nil {
			return err
		}
	} else {
		nodes = c.Node.NodeNet.GetNodes()
	}
	fmt.Println("Nodes:")

	for _, n := range nodes {
		fmt.Println("  ", n.NodeAddrToString())
	}

	return nil
}

// Add a node to connections
func (c *NodeCLI) commandAddNode() error {
	newaddr := lib.NodeAddr{c.Input.Args.NodeHost, c.Input.Args.NodePort}

	if c.AlreadyRunningPort > 0 {
		nc := c.getLocalNetworkClient()

		err := nc.SendAddNode(newaddr)

		if err != nil {
			return err
		}
	} else {
		c.Node.NodeNet.AddNodeToKnown(newaddr)
	}

	fmt.Println("Success!")

	return nil
}

// Remove a node from connections
func (c *NodeCLI) commandRemoveNode() error {
	remaddr := lib.NodeAddr{c.Input.Args.NodeHost, c.Input.Args.NodePort}
	fmt.Printf("Remove %s %d", c.Input.Args.NodeHost, c.Input.Args.NodePort)
	fmt.Println(remaddr)

	if c.AlreadyRunningPort > 0 {
		nc := c.getLocalNetworkClient()

		err := nc.SendRemoveNode(remaddr)

		if err != nil {
			return err
		}
	} else {
		c.Node.NodeNet.RemoveNodeFromKnown(remaddr)
	}
	fmt.Println("Success!")

	return nil
}
