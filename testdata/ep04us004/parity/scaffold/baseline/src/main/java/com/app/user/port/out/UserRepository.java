package com.app.user.port.out;

public interface UserRepository {

    Optional<User> findByEmail(String email);

    User save(User user);
}
