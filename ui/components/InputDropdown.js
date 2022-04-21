export default function InputDropdown ({
  label,
  type,
  value,
  placeholder,
  hasDropdownSelection = true,
  optionType,
  options,
  handleInputChange,
  handleSelectOption,
  handleKeyDown
}) {
  return (
    <div>
      {label &&
        <label htmlFor='price' className='block text-sm font-medium text-white'>
          {label}
        </label>}
      <div className='relative rounded shadow-sm'>
        <input
          type={type}
          value={value}
          className='block w-full px-4 py-3 sm:text-sm border border-gray-600 bg-transparent rounded-full focus:outline-none focus:ring focus:ring-cyan-600'
          placeholder={placeholder}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
        />
        {hasDropdownSelection &&
          <div className='absolute inset-y-0 right-2 flex items-center'>
            <label htmlFor={optionType} className='sr-only'>
              {optionType}
            </label>
            <select
              id={optionType}
              name={optionType}
              onChange={handleSelectOption}
              className='h-full py-0 pl-2 border-transparent bg-transparent text-gray-500 text-sm focus:outline-none'
            >
              {options.map((option) => (
                <option key={option} value={option}>{option}</option>
              ))}
            </select>
          </div>}
      </div>
    </div>
  )
}
