namespace IdentityService.Domain.Entities;



public sealed class UserState
{
    public required string Email { get; init; }
}

public sealed class User
{
    private User() { }

    public string Email { get; private set; } = string.Empty;
    public Guid Id { get; private set; }
    public Profile? Profile { get; private set; }

    public byte[] RowVersion { get; private set; } = [];

    public static User Create(UserState state) => new()
    {
        Id = Guid.NewGuid(),
        Email = state.Email,
    };

    public void Update(UserState state)
    {
        Email = state.Email;
    }
}
