from contract_support import run_contract
from decorator import evaluate
def test_shared_contract(): run_contract(__file__, evaluate)
