# Feature: x
Module: layered-module

## Contract: ContractB
Type: service
Methods:
- void run()

## Contract: ContractA
Type: service
DependsOn: ContractB
Methods:
- void execute()
