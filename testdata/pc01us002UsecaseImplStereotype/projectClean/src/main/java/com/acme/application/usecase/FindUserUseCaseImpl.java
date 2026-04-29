package com.acme.application.usecase;

import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;

@Service
@RequiredArgsConstructor
public class FindUserUseCaseImpl implements FindUserUseCase {

    private final UserRepository userRepository;

    @Override
    public User execute(Long id) {
        return userRepository.findById(id)
                .orElseThrow(() -> new IllegalArgumentException("User not found: " + id));
    }
}
