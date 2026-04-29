package com.acme.application.usecase;

import org.springframework.stereotype.Service;

@Service
public class FindUserUseCaseImpl implements FindUserUseCase {

    private final UserRepository userRepository;

    public FindUserUseCaseImpl(UserRepository userRepository) {
        this.userRepository = userRepository;
    }

    @Override
    public User execute(Long id) {
        return userRepository.findById(id)
                .orElseThrow(() -> new IllegalArgumentException("User not found: " + id));
    }
}
