//revive:disable
package sign

import (
	"bytes"
	"fmt"
	"image/jpeg"
	"strconv"
	"time"
)

// createVisualSignatureWithImage creates a visual signature field in a PDF document.
// visible: determines if the signature field should be visible or not.
// pageNumber: the page number where the signature should be placed.
// rect: the rectangle defining the position and size of the signature field.
// Returns the visual signature string and an error if any.
func (context *SignContext) createVisualSignatureWithImage(visible bool, pageNumber uint32, rect [4]float64) ([]byte, error) {
	var visual_signature bytes.Buffer

	visual_signature.WriteString("<<\n")

	// Define the object as an annotation.
	visual_signature.WriteString("  /Type /Annot\n")
	// Specify the annotation subtype as a widget.
	visual_signature.WriteString("  /Subtype /Widget\n")

	if visible {
		// Set the position and size of the signature field if visible.
		visual_signature.WriteString(fmt.Sprintf("  /Rect [%f %f %f %f]\n", rect[0], rect[1], rect[2], rect[3]))

		appearance, err := context.createAppearanceWithImage()
		if err != nil {
			return nil, fmt.Errorf("failed to create appearance: %w", err)
		}

		appearanceObjectId, err := context.addObject(appearance)
		if err != nil {
			return nil, fmt.Errorf("failed to add appearance object: %w", err)
		}

		// An appearance dictionary specifying how the annotation
		// shall be presented visually on the page (see 12.5.5, "Appearance streams").
		visual_signature.WriteString(fmt.Sprintf("  /AP << /N %d 0 R >>\n", appearanceObjectId))

	} else {
		// Set the rectangle to zero if the signature is invisible.
		visual_signature.WriteString("  /Rect [0 0 0 0]\n")
	}

	// Retrieve the root object from the PDF trailer.
	root := context.PDFReader.Trailer().Key("Root")
	// Get all keys from the root object.
	root_keys := root.Keys()
	found_pages := false
	for _, key := range root_keys {
		if key == "Pages" {
			// Check if the root object contains the "Pages" key.
			found_pages = true
			break
		}
	}

	// Get the pointer to the root object.
	rootPtr := root.GetPtr()
	// Store the root object reference in the catalog data.
	context.CatalogData.RootString = strconv.Itoa(int(rootPtr.GetID())) + " " + strconv.Itoa(int(rootPtr.GetGen())) + " R"

	if found_pages {
		// Find the page object by its number.
		page, err := context.findPageByNumber(pageNumber)
		if err != nil {
			return nil, err
		}

		// Get the pointer to the page object.
		page_ptr := page.GetPtr()

		// Store the page ID in the visual signature context so that we can add it to xref table later.
		context.VisualSignData.pageObjectId = page_ptr.GetID()

		// Add the page reference to the visual signature.
		visual_signature.WriteString("  /P " + strconv.Itoa(int(page_ptr.GetID())) + " " + strconv.Itoa(int(page_ptr.GetGen())) + " R\n")
	}

	// Define the annotation flags for the signature field (132)
	annotationFlags := AnnotationFlagPrint | AnnotationFlagLocked
	visual_signature.WriteString(fmt.Sprintf("  /F %d\n", annotationFlags))

	// Define the field type as a signature.
	visual_signature.WriteString("  /FT /Sig\n")
	// Set a unique title for the signature field.
	visual_signature.WriteString(fmt.Sprintf("  /T %s\n", pdfString("Signature "+strconv.Itoa(len(context.existingSignatures)+1))))

	// Reference the signature dictionary.
	visual_signature.WriteString(fmt.Sprintf("  /V %d 0 R\n", context.SignData.objectId))

	// Close the dictionary and end the object.
	visual_signature.WriteString(">>\n")

	return visual_signature.Bytes(), nil
}

func (context *SignContext) createAppearanceWithImage() ([]byte, error) {
	img := context.SignData.Appearance.Image
	dataBuf := bytes.Buffer{}
	err := jpeg.Encode(&dataBuf, img, &jpeg.Options{Quality: 100})
	if err != nil {
		return nil, err
	}
	data := dataBuf.Bytes()

	var img_buffer bytes.Buffer
	img_buffer.WriteString("<<\n")
	img_buffer.WriteString("  /Type /XObject\n")
	img_buffer.WriteString("  /Subtype /Image\n")
	fmt.Fprintf(&img_buffer, "  /Width %d\n", img.Bounds().Dx())
	fmt.Fprintf(&img_buffer, "  /Height %d\n", img.Bounds().Dy())
	img_buffer.WriteString("  /ColorSpace /DeviceRGB\n")
	img_buffer.WriteString("  /BitsPerComponent 8\n")
	img_buffer.WriteString("  /Filter /DCTDecode\n")
	fmt.Fprintf(&img_buffer, "  /Length %d\n", len(data))

	img_buffer.WriteString(">>\n")
	img_buffer.WriteString("stream\n")
	img_buffer.Write(data)
	img_buffer.WriteString("\nendstream\n")

	imgIdentifier := time.Now().UnixNano()
	imgBufferID, err := context.addObject(img_buffer.Bytes())
	if err != nil {
		return nil, err
	}

	var appearance_buffer bytes.Buffer
	appearance_buffer.WriteString("<<\n")
	appearance_buffer.WriteString("  /Type /XObject\n")
	appearance_buffer.WriteString("  /Subtype /Form\n")
	fmt.Fprintf(&appearance_buffer, "  /BBox [0 0 %d %d]\n", img.Bounds().Dx(), img.Bounds().Dy())
	fmt.Fprintf(&appearance_buffer, "  /Resources << /XObject << /Im%d %d 0 R >> >>", imgIdentifier, imgBufferID)
	appearance_buffer.WriteString("  /FormType 1\n")
	appearance_buffer.WriteString(">>\n")

	appearance_buffer.WriteString("stream\n")
	fmt.Fprintf(&appearance_buffer, "q %d 0 0 %d 0 0 cm\n", img.Bounds().Dx(), img.Bounds().Dy())
	fmt.Fprintf(&appearance_buffer, "/Im%d Do\n", imgIdentifier)
	appearance_buffer.WriteString("Q\n")
	appearance_buffer.WriteString("endstream\n")

	return appearance_buffer.Bytes(), nil
}
