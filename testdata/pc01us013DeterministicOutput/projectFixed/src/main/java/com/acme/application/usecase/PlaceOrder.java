package com.acme.application.usecase;

// PC01US-013 fixture: deliberately empty (no @Component, no @Service)
// so BOTH required_annotations rules fire, exercising the
// comparator's RuleID tiebreaker.
public class PlaceOrder {}
