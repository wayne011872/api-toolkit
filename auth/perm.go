package auth

type ApiPerm string

type ReqUser interface {
	GetHost() string
	GetPerms() []string
	GetId() string
	GetAccount() string
	GetName() string
	GetUsage() string
}

type reqUserImpl struct {
	host    string
	uid     string
	account string
	name    string
	roles   []string
	usage   string
}

func (u *reqUserImpl) GetHost() string {
	return u.host
}

func (u *reqUserImpl) GetPerms() []string {
	return u.roles
}

func (u *reqUserImpl) GetId() string {
	return u.uid
}

func (u *reqUserImpl) GetAccount() string {
	return u.account
}

func (u *reqUserImpl) GetName() string {
	return u.name
}

func (u *reqUserImpl) GetUsage() string {
	return u.usage
}
