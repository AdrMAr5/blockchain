from chain import Chain

chain = Chain(20)

i = 0

while(True):
    data = input("Add something to the chain: ")
    chain.add_to_pool(data)
    chain.mine()
    print(chain.blocks[i])
    i += 1