package emcstorage

//GetPool purpose is to get information about pool in the storage system. .
// "description" добавлено в список запрашиваемых полей 2026-09-16 по
// запросу пользователя loadmon "для VNX/Unity/Lenovo тоже нужно" (обработка
// пулов, обязательно с полем description) - см. PoolContent.Description и
// cmdb.syncUnityHardware.
func (session *Session) GetPool() (resp *Pool, err error) {

	fields := "id,name,description,sizeFree,sizeTotal,sizeUsed,sizeSubscribed,health"

	err = session.Request("GET", "/api/types/pool/instances", fields, "", false, nil, &resp)
	return resp, err
}

//GetStorageResource purpose is to get information about storage resources in the storage system.
func (session *Session) GetStorageResource() (resp *StorageResource, err error) {

	fields := "id,name,sizeAllocated,sizeTotal,sizeUsed"

	err = session.Request("GET", "/api/types/storageResource/instances", fields, "", false, nil, &resp)
	return resp, err
}
