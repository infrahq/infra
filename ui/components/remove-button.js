import { useState } from 'react'

import DeleteModal from './delete-modal'

export default function RemoveButton({
  children = 'Remove',
  onRemove,
  modalTitle,
  modalMessage,
}) {
  const [modalOpen, setModalOpen] = useState(false)

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
