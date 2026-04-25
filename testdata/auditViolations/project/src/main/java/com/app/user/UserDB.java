package com.app.user;

import jakarta.persistence.Entity;

// Violation: @Entity outside domain/ — triggers annotation_path_mismatch (entity-path-mismatch)
@Entity
public class UserDB {
    private Long id;
    private String name;
}
