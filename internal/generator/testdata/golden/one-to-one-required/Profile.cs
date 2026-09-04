namespace IdentityService.Domain.Entities;



public sealed class ProfileState
{
    public required string DisplayName { get; init; }
    public required Guid UserId { get; init; }
}

public sealed class Profile
{
    private Profile() { }

    public string DisplayName { get; private set; } = string.Empty;
    public Guid Id { get; private set; }
    public Guid UserId { get; private set; }
    public User User { get; private set; } = null!;

    public byte[] RowVersion { get; private set; } = [];

    public static Profile Create(ProfileState state) => new()
    {
        Id = Guid.NewGuid(),
        DisplayName = state.DisplayName,
        UserId = state.UserId,
    };

    public void Update(ProfileState state)
    {
        DisplayName = state.DisplayName;
        UserId = state.UserId;
    }
}
