package mongo

var (
	Users = &DefaultCollection{"users"}
	Sessions = &DefaultCollection{"sessions"}
	Answers = &DefaultCollection{"answers"}
	Units = &DefaultCollection{"units"}
)

type DefaultCollection struct {
	CName string
}

func (d *DefaultCollection) Count(query interface{}) (int, error) {
	s := GetSessionCopy()
	defer s.Close()
	return s.Find(d.CName, query).Count()
}

func (d *DefaultCollection) Insert(query interface{}) error {
	s := GetSessionCopy()
	defer s.Close()
	return s.Insert(d.CName, query)
}

func (d *DefaultCollection) Remove(selector interface{}) error {
	s := GetSessionCopy()
	defer s.Close()
	return s.Remove(d.CName, selector)
}

func (d *DefaultCollection) FindAll(query interface{}, result interface{}) error {
	s := GetSessionCopy()
	err := s.Find(d.CName, query).All(result)
	s.Close()
	return err
}

func (d *DefaultCollection) FindOne(query interface{}, result interface{}) error {
	s := GetSessionCopy()
	defer s.Close()
	return s.Find(d.CName, query).One(result)
}

func (d *DefaultCollection) Update(selector, update interface{}) error {
	s := GetSessionCopy()
	defer s.Close()
	return s.Update(d.CName, selector, update)
}

func (d *DefaultCollection) Upsert(selector, update interface{}) error {
	s := GetSessionCopy()
	defer s.Close()
	return s.Upsert(d.CName, selector, update)
}