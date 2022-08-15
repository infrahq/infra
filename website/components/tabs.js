import { useState, createContext } from 'react'

export const TabContext = createContext()

export default function Tabs({ labels, children }) {
  const [currentTab, setCurrentTab] = useState(labels[0])

  return (
    <TabContext.Provider value={currentTab}>
      <div className='flex flex-wrap space-x-2'>
        {labels.map(label => (
          <button
            key={label}
            className={`${
              label === currentTab
                ? ' bg-blue-600/5 text-blue-700 hover:text-blue-700'
                : 'text-gray-500 hover:text-gray-800'
            } rounded-lg px-7 py-1.5 text-sm font-medium leading-loose first-of-type:ml-0`}
            onClick={() => setCurrentTab(label)}
          >
            {label}
          </button>
        ))}
      </div>
      {children}
    </TabContext.Provider>
  )
}
