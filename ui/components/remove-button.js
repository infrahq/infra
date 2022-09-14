import { useState } from 'react'

import DeleteModal from './delete-modal'

export default function RemoveButton({
  children = 'Remove',
  deleteModalType = 'sm',
  onRemove,
  modalTitle,
  modalMessage,
}) {
  const [modalOpen, setModalOpen] = useState(false)

  return (
    <>
      <button
        data-testid='remove-button'
        type='button'
        onClick={() => setModalOpen(true)}
        className='inline-flex items-center rounded-md border border-gray-300 bg-white px-4 py-2 text-xs font-medium text-gray-700 shadow-sm hover:bg-gray-100'
      >
        {children}
      </button>
      <DeleteModal
        type={deleteModalType}
        open={modalOpen}
        setOpen={setModalOpen}
        onSubmit={() => {
          onRemove()
          setModalOpen(false)
        }}
        title={modalTitle}
        message={modalMessage}
      />
    </>
  )
}
