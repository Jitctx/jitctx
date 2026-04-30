package com.acme.application.usecase;

public interface UseCase<I, O> {
    O execute(I input);
}
