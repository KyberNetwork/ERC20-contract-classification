var tracer = {
  sloads: [],
  step: function (log, db) {
    let op = log.op.toNumber()
    if (op == 0x54 /*SLOAD*/ || op == 0x55 /*SSTORE*/ ) {
      let addr = log.contract.getAddress()
      let slot = toWord(log.stack.peek(0).toString(16))
      this.sloads.push({
        op: op,
        addr: toHex(addr),
        slot: toHex(slot),
        value: toHex(db.getState(addr, slot))
      })
    }

  },
  result: function (ctx) {
    return {
      sloads: this.sloads,
      output: toHex(ctx.output)
    }
  },
  fault: function () { }
}