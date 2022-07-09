import DeleteModal from './delete-modal'

export default function Remove({
  buttonText = 'Remove',
  deleteModalOpen,
  setDeleteModalOpen,
  onSubmit,
  deleteModalTitle,
  deleteModalMessage,
}) {
  return (
    <>
      <button
        type='button'
        onClick={() => setDeleteModalOpen(true)}
        className='flex items-center rounded-md border border-violet-300 px-6 py-3 text-2xs text-violet-100'
      >
        {buttonText}
      </button>
      <DeleteModal
        open={deleteModalOpen}
        setOpen={setDeleteModalOpen}
        onSubmit={() => onSubmit()}
        title={deleteModalTitle}
        message={deleteModalMessage}
      />
    </>
  )
}
