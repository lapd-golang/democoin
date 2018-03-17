import _lib
import _transfers
import re
import time
import startnode
import transactions

#def beforetest(testfilter):
#    print "before test"
#def aftertest(testfilter):
#    print "after test"
def test(testfilter):
    _lib.StartTestGroup("Blocks making")

    _lib.CleanTestFolders()
    datadir = _lib.CreateTestFolder()

    startnode.StartNodeWithoutBlockchain(datadir)
    address = startnode.InitBockchain(datadir)
    startnode.StartNode(datadir, address, '30000')
    startnode.StopNode(datadir)
    
    # create another 3 addresses
    address2 = transactions.CreateWallet(datadir)
    address3 = transactions.CreateWallet(datadir)

    startnode.StartNode(datadir, address, '30000')

    _lib.StartTestGroup("Do transactions")

    transactions.GetUnapprovedTransactionsEmpty(datadir)
    
    amount1 = '1'
    amount2 = '2'
    amount3 = '3'
    
    txid1 = _transfers.Send(datadir,address,address2,amount1)
    
    txlist = transactions.GetUnapprovedTransactions(datadir)
    
    _lib.FatalAssert(len(txlist) == 1,"Should be 1 unapproved transaction")
    
    #block making will be started now 
    time.sleep(4)
        
    txid2 = _transfers.Send(datadir,address,address3,amount2)
    
    txlist = transactions.GetUnapprovedTransactions(datadir)
    
    _lib.FatalAssert(len(txlist) == 1,"Should be 1 unapproved transaction")
    
    txid3 = _transfers.Send(datadir,address,address3,amount3)
    
    # node needs some time to make a block, so transaction still will be in list of unapproved
    txlist = transactions.GetUnapprovedTransactions(datadir)
    
    _lib.FatalAssert(len(txlist) == 2,"Should be 2 unapproved transaction")
    
    if txid2 not in txlist.keys():
        _lib.Fatal("Transaction 2 is not in the list of transactions")
    
    if txid3 not in txlist.keys():
        _lib.Fatal("Transaction 3 is not in the list of transactions")
    
    _lib.FatalAssertFloat(amount2, txlist[txid2][2], "Amount of transaction 2 is wrong")
    
    _lib.FatalAssertFloat(amount3, txlist[txid3][2], "Amount of transaction 3 is wrong")
    
    time.sleep(5)
    
    transactions.GetUnapprovedTransactionsEmpty(datadir)
    
    startnode.StopNode(datadir)
    
    #_lib.RemoveTestFolder(datadir)
    _lib.EndTestGroupSuccess()
    