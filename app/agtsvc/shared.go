package agtsvc

type SharedService interface{}

type sharedService struct{}

func (biz *sharedService) StringsSet() {
}
