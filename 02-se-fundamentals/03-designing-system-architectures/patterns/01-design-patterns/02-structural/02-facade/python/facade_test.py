from contract_support import run_contract
from facade import evaluate
def test_shared_contract(): run_contract(__file__, evaluate)
