package com.acme.application.usecase;

public interface UseCase<I> {
    Object execute(I input);
}
