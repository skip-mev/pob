## Questions
- Should probably make the Unit object a struct? Reason why is because we want some common things to also exist on the structs like antehandlers which will simply define the validation logic for the unit.
- need to figure out how to handle the creation of the block. 
- would be nice to have options for how you want to configure the mempool such that you can inherit a default mempool and then override certain things.
- maybe a nice feature if you only include the message types you want registered instead of matching it with an antehandler.
- building the block using exclusively hooks in prepare and process proposal is problematic because some transactions will use some ante handlers and not others so we cant just recheck the entire block.