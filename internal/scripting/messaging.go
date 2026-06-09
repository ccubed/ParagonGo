package scripting

var (
	disableMessageQueue = false
)

func setMessagingFunctions(vm registrar) {

	vm.Set(`console`, consoleObject())
	vm.Set(`SendUserMessage`, SendUserMessage)
	vm.Set(`SendRoomMessage`, SendRoomMessage)
	vm.Set(`SendRoomExitsMessage`, SendRoomExitsMessage)
	vm.Set(`SendBroadcast`, SendBroadcast)

}
