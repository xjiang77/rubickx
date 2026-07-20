from contract_support import run_contract
from fan_out_fan_in import evaluate

def test_shared_contract():
    run_contract(__file__, evaluate)
