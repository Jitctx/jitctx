package com.app.user.domain;

import org.springframework.stereotype.Component;
import com.app.user.adapter.persistence.UserRepository;

// Violation 1: domain/ file imports org.springframework.* — triggers domain-leak
// Violation 2: domain/ file imports from .adapter. package — triggers domain-adapter-inversion
// The config.yaml disables domain-leak; only domain-adapter-inversion appears in the report.
public class User {
    private Long id;
    private String name;
    private String email;
}
