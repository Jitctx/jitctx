# Feature: create-user
Module: user-management
Package: com.app.user

## Contract: CreateUserUseCase
Type: input-port
Methods:
- UserResponse execute(CreateUserCommand cmd)

## Contract: UserRepository
Type: output-port
Methods:
- Optional<User> findByEmail(String email)
- User save(User user)

## Contract: UserServiceImpl
Type: service
Implements: CreateUserUseCase
DependsOn: UserRepository

## Contract: UserController
Type: rest-adapter
Uses: CreateUserUseCase
Endpoints:
- POST /users

## Contract: User
Type: aggregate-root
