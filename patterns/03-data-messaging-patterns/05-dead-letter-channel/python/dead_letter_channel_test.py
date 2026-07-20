from contract_support import run_contract
from dead_letter_channel import evaluate

def test_shared_contract():
    run_contract(__file__, evaluate)
