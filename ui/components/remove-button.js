import DeleteModal from './delete-modal'

export default function RemoveButton({
  children = 'Remove',
  modalOpen,
  setModalOpen,
  onSubmit,
  modalTitle,
  modalMessage,
}) {
  return (
    <>
      <button
        type='button'
        onClick={() => setModalOpen(true)}
        className='flex items-center rounded-md border border-violet-300 px-6 py-3 text-2xs text-violet-100'
      >
        {children}
      </button>
      <DeleteModal
        open={modalOpen}
        setOpen={setModalOpen}
        onSubmit={() => onSubmit()}
        title={modalTitle}
        message={modalMessage}
      />
    </>
  )
}
